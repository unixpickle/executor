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
	dir, _ := os.Getwd()
	args := os.Args[1:]
	
	// Create the command
	cfg := new(executor.Command)
	cfg.Directory = dir
	cfg.Arguments = args
	cfg.Environment = map[string]string{}
	
	// Run the relauncher
	rl := executor.Relaunch(cfg.ToJob(), time.Second)
	fmt.Println("hit enter to stop the relauncher...")
	fmt.Scanln()
	rl.Stop()
}

