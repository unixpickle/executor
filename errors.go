package executor

import "errors"

// These are errors which may be returned from various executable functions.
var (
	ErrAlreadyRunning = errors.New("Job already running.")
	ErrNotRunning     = errors.New("Job is not running.")
	ErrNotWaiting     = errors.New("Job is not waiting to relaunch.")
)
