package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
	"github.com/vishvananda/netlink"
)

// the two halves of docker's -p 3000:3000
const (
	containerPort = "3000"
	hostPort      = "3000"
	rootfs        = "rootfs"
	veth1         = "veth1"
	veth2         = "veth2"
)

func main() {
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
	fmt.Println("running cmd:", cmdName, "with args:", args)

	// execute ourself
	cmd := exec.Command("/proc/self/exe", append([]string{"reexec", cmdName}, args...)...)

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
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{{
			// the userid in the container
			ContainerID: 0,
			// mapped to our current uid on host
			HostID: os.Getuid(),
			// just map one uid
			Size: 1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      os.Getgid(),
			Size:        1,
		}},
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
	createParentVeth(pid)

	// close writer sends eof to cprocess
	must("close writer to send cp eof signal", w.Close())

	fmt.Println("child pid: ", pid)
	if err := cmd.Wait(); err != nil {
		fmt.Println("error occured while waiting for process to finish", err)
	}
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

	must("change hostname", syscall.Sethostname([]byte("container")))
	must("change root", syscall.Chroot(rootfs))
	must("change dir to root level", syscall.Chdir("/"))

	// mount a new proc
	must("mount /proc", syscall.Mount("proc", "/proc", "proc", 0, ""))

	path, err := exec.LookPath(cmdName)
	must("resolve command path", err)

	must("execve the new process", syscall.Exec(path, append([]string{cmdName}, args...), os.Environ()))
}

func createParentVeth(cpid int) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: veth1,
		},
		PeerName:      veth2,
		PeerNamespace: netlink.NsPid(cpid),
	}
	// create the link from parent to child
	must("create veth", netlink.LinkAdd(veth))

	// get the link
	link, err := netlink.LinkByName(veth1)
	must("resolve veth1 link", err)

	// set addr
	addr, err := netlink.ParseAddr("10.0.0.1/24")
	must("parse addr ", err)
	must("add addr ", netlink.AddrAdd(link, addr))
	// enable device
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

	fmt.Printf("%s %s\n", what, sb.String())
	fmt.Println("exiting")
	os.Exit(1)
}
