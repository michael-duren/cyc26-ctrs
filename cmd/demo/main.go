package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
)

const (
	// 	containeraddr = "10.0.0.2"
	// 	port          = "3000"
	rootfs = "_rootfs"

	// veth1         = "veth1"
	// veth2         = "veth2"
	// cgrouppath    = "/sys/fs/cgroup/user.slice/user-1000.slice/boxes.service"
	// ctrpath       = cgrouppath + "/ctr1"
	// fileperms     = 0o755
	uid = 1000
	gid = 1000
)

var (
	env = []string{
		"USER=root",
		"HOME=/root",
		"SHELL=/usr/bin/bash",
		"PATH=/bin:/sbin:/usr/local/sbin:/usr/local/bin:/usr/bin",
		"TERM=xterm-256color",
	}
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

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER,
		// map root uid, gid to a non root user on the 'outside' or host
		UidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      uid,
			Size:        1,
		}},
		GidMappings: []syscall.SysProcIDMap{{
			ContainerID: 0,
			HostID:      gid,
			Size:        1,
		}},
		// sets the user the child process starts as, in this case root
		Credential: &syscall.Credential{Uid: 0, Gid: 0},
	}

	// hook up process stdin to ours
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("error: ", err)
		os.Exit(1)
	}
}

func reexec(cmdName string, args []string) {
	fmt.Println("reexecing cmd:", cmdName, "with args:", args)

	must("make mounts private", syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""))

	must("change root", syscall.Chroot(rootfs))
	must("change to new root", syscall.Chdir("/"))

	must("mount proc", syscall.Mount("proc", "/proc", "proc", 0, ""))

	must("change hostname", syscall.Sethostname([]byte("container")))
	must("exev the new process", syscall.Exec(cmdName, args, env))
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
