package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
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
}

func reexec(cmdName string, args []string) {
	fmt.Println("reexecing cmd:", cmdName, "with args:", args)
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
