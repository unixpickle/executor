package executor

// Job is a restartable task which can be run synchronously.
type Job interface {
	// Start starts the job. After this is called, calling Stop() must stop the
	// job.
	// This is not thread-safe.
	Start() error

	// Stop stops the job asynchronously if it is running.
	// This is thread-safe.
	Stop() error

	// Wait waits for the job to finish.
	// This is not thread-safe.
	Wait() error
}
