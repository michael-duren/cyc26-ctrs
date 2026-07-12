package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
)

func main() {
	if len(os.Args) < 3 {
		helpers.Usage()
		return
	}
	cmd := os.Args[1]
	ctrCmd := os.Args[2]
	cmdArgs := os.Args[3:]
	switch cmd {
	case "run":
		run(ctrCmd, cmdArgs)
	case "reexec":
		reexec(ctrCmd, cmdArgs)
	default:
		helpers.Usage()
	}
}

// run is the host-side half: it re-invokes this same binary as
// "demo reexec ..." in fresh namespaces. The namespace flags only apply to
// the child, so all the jailing happens over in reexec.
func run(cmdName string, args []string) {
	fmt.Println("running cmd: ", cmdName, "with args: ", args)

	reexecArgs := append([]string{"reexec", cmdName}, args...)
	cmd := exec.Command("/proc/self/exe", reexecArgs...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// NEWNS: own mount table, NEWUTS: own hostname, NEWPID: child is PID 1
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}

	if err := cmd.Run(); err != nil {
		fmt.Println("error: ", err)
		os.Exit(1)
	}
}

// reexec is the container-side half, already inside the new namespaces:
// jail the process into ./rootfs, mount /proc so ps/top work, then execve
// the user's command so it takes over as PID 1.
func reexec(cmdName string, args []string) {
	fmt.Println("reexecing cmd: ", cmdName, "with args: ", args)

	must("set hostname", syscall.Sethostname([]byte("demo")))

	// Our new mount namespace starts as a copy of the host's, and on systemd
	// systems those mounts are "shared" — without this, the /proc mount below
	// would propagate back to the host and outlive us.
	must("make mounts private", syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""))

	must("chroot", syscall.Chroot("rootfs"))
	// chroot does NOT move the cwd inside the jail — skipping this is the
	// classic chroot escape.
	must("chdir into jail", syscall.Chdir("/"))
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
