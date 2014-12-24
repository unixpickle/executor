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
	dir, _ := os.Getwd()
	args := os.Args[1:]
	cfg := new(executor.Command)
	cfg.Directory = dir
	cfg.Arguments = args
	cfg.Environment = map[string]string{}
	job := cfg.ToJob()
	job.Start()
	job.Wait()
}

