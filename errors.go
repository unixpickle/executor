package executor

import "errors"

var (
	ErrAlreadyRunning = errors.New("Executable already running.")
	ErrNotRunning     = errors.New("Executable is not running.")
	ErrNotWaiting     = errors.New("Task is not waiting to relaunch.")
)
