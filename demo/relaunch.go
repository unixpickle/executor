package main

import (
	"fmt"
	"github.com/unixpickle/executor"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: relaunch <arguments ...>")
		os.Exit(1)
	}
	
	// Create the command
	cmd := executor.Command(os.Args[1:]...)
	cmd.Directory, _ = os.Getwd()
	
	// Run the relauncher
	rl := executor.Relaunch(cmd.ToJob(), time.Second)
	fmt.Println("hit enter to stop the relauncher...")
	fmt.Scanln()
	rl.Stop()
}

