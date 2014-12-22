package executor

import "errors"

var (
	ErrAlreadyRunning = errors.New("Executable already running.")
	ErrNotRunning     = errors.New("Executable is not running.")
)

