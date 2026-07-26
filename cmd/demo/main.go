package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
)

// the two halves of docker's -p 3000:3000
const (
	containerPort = "3000"
	hostPort      = "3000"
	rootfs        = "rootfs"
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
	fmt.Println("running cmd:", cmdName, "with args:", args)

	cmd := exec.Command(cmdName, args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// clone flags are the arguments we can pass to the `clone`
		// syscall to configure our new process namespaces
		// newuts specifically allows us to reset the hostname, fun fact
		// has nothing to do with keeping track of time
		// TODO: Add new process ns
		Cloneflags: syscall.CLONE_NEWUTS,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// TODO: Start wiring this into reexec, and mount proc
	must("change hostname", syscall.Sethostname([]byte("container")))
	must("change root", syscall.Chroot(rootfs))
	must("change dir to root level", syscall.Chdir("/"))

	must("run users command and args", cmd.Run())
}

func reexec(cmdName string, args []string) {
	fmt.Println("running cmd:", cmdName, "with args:", args)
	fmt.Println("I'm in reexec", containerPort, hostPort)
}

func must(what string, err error) {
	if err != nil {
		fmt.Println(what+" error: ", err)
		os.Exit(1)
	}
}
