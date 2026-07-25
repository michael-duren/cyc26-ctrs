package main

import (
	"fmt"

	"github.com/michael-duren/boxes/presentation-project/internal/helpers"
)

// the two halves of docker's -p 3000:3000
const (
	containerPort = "3000"
	hostPort      = "3000"
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
}

func reexec(cmdName string, args []string) {
	fmt.Println("running cmd:", cmdName, "with args:", args)
	fmt.Println("I'm in reexec", containerPort, hostPort)
}

// func must(what string, err error) {
// 	if err != nil {
// 		fmt.Println(what+" error: ", err)
// 		os.Exit(1)
// 	}
// }
