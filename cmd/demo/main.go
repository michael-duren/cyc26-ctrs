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

	must("run users command and args", cmd.Run())
}

func reexec(cmdName string, args []string) {
	fmt.Println("reexecing cmd:", cmdName, "with args:", args)
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

func must(what string, err error) {
	if err != nil {
		fmt.Println(what+" error: ", err)
		os.Exit(1)
	}
}
