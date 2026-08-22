// a simple package of helpers for the presentation
package helpers

import (
	"fmt"
	"os"
)

func Usage() {
	fmt.Println(`
About:
		A basic containerized process for educational purposes, runs the rootfs located in the working directory from where the command was run
Usage:
        demo <command> [arguments]
Commands:
        run         run the command and arguments
Example:
	  demo run /bin/bash         Runs '/bin/bash' in a containerized process
		`)
}

type CmdInput struct {
	RuntimeCmd   string
	ContainerCmd string
	CmdArgs      []string
}

func ParseInput() (*CmdInput, error) {
	if len(os.Args) < 3 {
		Usage()
		os.Exit(2)
	}

	return &CmdInput{
		RuntimeCmd:   os.Args[1],
		ContainerCmd: os.Args[2],
		CmdArgs:      os.Args[3:],
	}, nil
}
