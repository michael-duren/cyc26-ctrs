package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
	"github.com/michael-duren/boxes/presentation-project/internal/proxy"
	"github.com/michael-duren/boxes/presentation-project/internal/root"
	"github.com/vishvananda/netlink"
)

const (
	containeraddr = "10.0.0.2"
	port          = "3000"
	rootfs        = "_rootfs"
	veth1         = "veth1"
	veth2         = "veth2"
	cgrouppath    = "/sys/fs/cgroup/user.slice/user-1000.slice/boxes.service"
	ctrpath       = cgrouppath + "/ctr1"
	fileperms     = 0o755
	uid           = 1000
	gid           = 1000
)

func main() {
	defer die()
	cmdInput, err := helpers.ParseInput()
	must("have right cmds and args", err)
	switch cmdInput.RuntimeCmd {
	case "run":
		run(cmdInput.ContainerCmd, cmdInput.CmdArgs)
	case "reexec":
		reexec(cmdInput.ContainerCmd, cmdInput.CmdArgs)
	default:
		helpers.Usage()
	}
}

func run(cmdName string, args []string) {
	defer die()
	fmt.Println("running cmd:", cmdName, "with args:", args)

	cmd := exec.Command("/proc/self/exe", append([]string{"reexec", cmdName}, args...)...)

	cg()
	defer func() { must("cleanup cg", cleannupcg()) }()
	// get fd for cgroup
	cfd, err := syscall.Open(ctrpath, syscall.O_RDONLY|syscall.O_DIRECTORY, 0)
	must("open new cgroup", err)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// clone flags are the arguments we can pass to the `clone`
		// syscall to configure our new process namespaces
		// newuts specifically allows us to reset the hostname, fun fact
		// has nothing to do with keeping track of time
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			// the mount ns was the first and was just called ns originally
			// mount ns gives us separate mount tables ESSENTIAL for being able
			// to safely mount a new /proc fs, which is needed for process isolation
			// and view in the container
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWCGROUP,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{{
			// the userid in the container
			ContainerID: 0,
			// mapped to our current uid on host
			HostID: uid,
			// just map one uid
			Size: 1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      gid,
			Size:        1,
		}},
		Credential:  &syscall.Credential{Uid: 0, Gid: 0},
		UseCgroupFD: true,
		CgroupFD:    cfd,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r, w, err := os.Pipe()
	must("open pipe", err)

	// cmd.ExtraFiles =
	cmd.ExtraFiles = []*os.File{r}
	must("start container process", cmd.Start())

	pid := cmd.Process.Pid
	defer cleanupVEth()
	createParentVeth(pid)

	// close writer sends eof to cprocess
	must("close writer to send cp eof signal", w.Close())

	go func() {
		// docker-style -p: wildcard bind covers localhost, 10.0.0.1, and LAN
		err := proxy.Proxy(proxy.ToAddr("0.0.0.0", port), proxy.ToAddr(containeraddr, port))
		if err != nil {
			fmt.Println("err proxying", err)
		}
	}()

	must("wait for container process to finish", cmd.Wait())
}

func reexec(cmdName string, args []string) {
	fmt.Println("reexecing cmd:", cmdName, "with args:", args)
	// wait for parent setup to finish
	r := os.NewFile(3, "sync")
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if !errors.Is(err, io.EOF) {
		must("receive eof from parent", err)
	}

	createChildVeth()
	fmt.Println("after createchildveth")

	// we're not re mounting anything just setting all mounts to be private so that changes
	// aren't shared with parent ns i.e. host
	must("make mounts private", syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""))

	// docker-style hostname: random 12 hex chars (docker uses the container id prefix)
	hn := make([]byte, 6)
	_, err = rand.Read(hn)
	must("generate hostname", err)
	must("change hostname", syscall.Sethostname([]byte(hex.EncodeToString(hn))))
	// must("change root", syscall.Chroot(rootfs))
	// must("change dir to root level", syscall.Chdir("/"))
	must("pivot root", root.PivotRoot(rootfs, fileperms))

	// mount a new proc
	must("mount /proc", syscall.Mount("proc", "/proc", "proc", 0, ""))
	// mount a new cgroup
	must("make cgroup dir", os.MkdirAll("/sys/fs/cgroup", fileperms))
	must("mount /sys/fs/cgroup", syscall.Mount("cgroup2", "/sys/fs/cgroup", "cgroup2", 0, ""))

	path, err := exec.LookPath(cmdName)
	must("resolve command path", err)

	env := []string{
		"USER=root",
		"HOME=/root",
		"SHELL=/usr/bin/bash",
		"PATH=/bin:/sbin:/usr/local/sbin:/usr/local/bin:/usr/bin",
		"TERM=xterm-256color",
	}
	must("execve the new process", syscall.Exec(path, append([]string{cmdName}, args...), env))
}

func cg() {
	must("create service cgroup", os.MkdirAll(cgrouppath, fileperms))
	must("enable pids controller", os.WriteFile(
		filepath.Join(cgrouppath, "cgroup.subtree_control"), []byte("+pids"), fileperms))

	must("create ctr cgroup", os.MkdirAll(ctrpath, fileperms))
	must("write pids.max", os.WriteFile(filepath.Join(ctrpath, "pids.max"), []byte("50"), fileperms))
}

func cleannupcg() error {
	fmt.Println("running in cleanup")
	ctrpath := filepath.Join(cgrouppath, "ctr1")

	if err := os.Remove(ctrpath); err != nil {
		return err
	}

	return os.Remove(cgrouppath)
}

func createParentVeth(cpid int) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: veth1,
		},
		PeerName:      veth2,
		PeerNamespace: netlink.NsPid(cpid),
	}

	cleanupVEth()
	must("create veth", netlink.LinkAdd(veth))

	link, err := netlink.LinkByName(veth1)
	must("resolve veth1 link", err)

	addr, err := netlink.ParseAddr("10.0.0.1/24")
	must("parse addr ", err)
	must("add addr ", netlink.AddrAdd(link, addr))
	must("set link to up", netlink.LinkSetUp(link))
}

// createChildVeth assumed [createParentVeth] has been called
// finishes setting up the veth connection from the pov of
// the child
func createChildVeth() {
	link, err := netlink.LinkByName(veth2)
	must("resolve netlink from veth2", err)
	addr, err := netlink.ParseAddr("10.0.0.2/24")
	must("parse addr", err)
	must("add addr veth2", netlink.AddrAdd(link, addr))
	must("set veth2 ↑", netlink.LinkSetUp(link))

	// NOTE: child p loopback is down by default node process would fail
	link, err = netlink.LinkByName("lo")
	must("resolve link from loopback", err)
	must("set loopback up", netlink.LinkSetUp(link))
}

func must(what string, errs ...error) {
	errSlice := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			errSlice = append(errSlice, err)
		}
	}

	if len(errSlice) == 0 {
		return
	}
	var sb strings.Builder

	for _, err := range errSlice {
		fmt.Fprintf(&sb, "%s ", err.Error())
	}

	panic(fmt.Sprintf("%s %s", what, sb.String()))
}

func die() {
	// recover panic and exit 1 to not leak stack trace
	if r := recover(); r != nil {
		fmt.Println(r)
		os.Exit(1)
	}
}

func cleanupVEth() {
	link, err := netlink.LinkByName(veth1)
	if err != nil {
		terr, ok := errors.AsType[netlink.LinkNotFoundError](err)
		if ok {
			return
		}
		fmt.Println("cleanup: lookup veth1", err, terr)
		return
	}
	if err := netlink.LinkDel(link); err != nil {
		fmt.Println("cleanup: deleting veth1", err)
	}
}
