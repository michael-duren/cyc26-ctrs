package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
)

func main() {
	cmdInput := helpers.ParseInput()
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
	fmt.Println("running cmd: ", cmdName, "with args: ", args)

	// create new process of program
	reexecArgs := append([]string{"reexec", cmdName}, args...)
	cmd := exec.Command("/proc/self/exe", reexecArgs...)

	// set childs io to same OFD as parent process
	// so that we can easily interact during presentation
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// new process has the new namespaces
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}

	// cmd.run actually calls copy to create new process
	// with out namespaces
	if err := cmd.Run(); err != nil {
		fmt.Println("error: ", err)
		os.Exit(1)
	}
}

func reexec(cmdName string, args []string) {
	fmt.Println("reexecing cmd: ", cmdName, "with args: ", args)

	must("set hostname", syscall.Sethostname([]byte("demo")))

	// Our new mount namespace starts as a copy of the host's, and on systemd
	// systems those mounts are "shared" without this, the /proc mount below
	// would propagate back to the host and outlive us.
	must("make mounts private", syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""))

	must("chroot", syscall.Chroot("rootfs"))

	must("chdir into jail", syscall.Chdir("/"))
	// completely isolate child processes
	must("mount /proc", syscall.Mount("proc", "/proc", "proc", 0, ""))

	bin, err := exec.LookPath(cmdName)
	must("find "+cmdName, err)

	// execve replaces this process image with the command, keeping our PID —
	// so the user's command runs as PID 1 inside the container.
	must("exec", syscall.Exec(bin, append([]string{cmdName}, args...), os.Environ()))
}

func must(what string, err error) {
	if err != nil {
		fmt.Println(what+" error: ", err)
		os.Exit(1)
	}
}
