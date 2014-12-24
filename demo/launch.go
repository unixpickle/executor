package main

import (
	"fmt"
	"github.com/unixpickle/executor"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: launch <arguments ...>")
		os.Exit(1)
	}
	
	// Create the command
	cmd := executor.Command(os.Args[1:]...)
	cmd.Directory, _ = os.Getwd()
	
	job := cmd.ToJob()
	job.Start()
	job.Wait()
}

