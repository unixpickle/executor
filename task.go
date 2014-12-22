package executor

import (
	"sync"
	"time"
)

// History records the history of a Task including its last launch,
// exit, and error.
type History struct {
	LastStart time.Time `json:"last_start"`
	LastStop  time.Time `json:"last_stop"`
	LastError time.Time `json:"last_error"`
	Error     string    `json:"error"`
}

// Status records the running status of a task.
type Status struct {
	Executing bool `json:"executing"`
	Done      bool `json:"done"`
}

// Task stores runtime information for a task.
type Task struct {
	mutex   sync.RWMutex
	history History
	status  Status

	onDone   chan struct{}
	stop     chan struct{}
	skipWait chan struct{}
}

// StartTask begins the execution loop for a given task.
func StartTask(config *Config) *Task {
	res := &Task{sync.RWMutex{}, History{}, Status{}, make(chan struct{}),
		make(chan struct{}, 1), make(chan struct{})}
	go res.loop(config.Clone())
	return res
}

// History returns the task's history.
func (t *Task) History() History {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.history
}

// HistoryStatus returns both the task's history and it's status.
func (t *Task) HistoryStatus() (History, Status) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.history, t.status
}

// SkipWait skips the current relaunch timeout if applicable.
// If the task was not relaunching, this returns ErrNotWaiting.
func (t *Task) SkipWait() error {
	select {
	case t.skipWait <- struct{}{}:
		return nil
	default:
		return ErrNotWaiting
	}
}

// Status returns the task's status
func (t *Task) Status() Status {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.status
}

// Stop stops the task.
// This method waits for the background process and Goroutines to terminate.
func (t *Task) Stop() {
	// Send async stop message
	select {
	case t.stop <- struct{}{}:
	default:
	}
	<-t.onDone
}

// Wait waits for the task to stop entirely.
func (t *Task) Wait() {
	<-t.onDone
}

func (t *Task) loop(c *Config) {
	for !t.stopped() {
		if !t.run(c) {
			break
		}
		if !c.Relaunch {
			break
		}
		if !t.restart(c) {
			break
		}
	}
	t.mutex.Lock()
	t.status.Done = true
	t.mutex.Unlock()
	close(t.onDone)
}

func (t *Task) reportError(err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.history.LastError = time.Now()
	t.history.Error = err.Error()
}

func (t *Task) reportStart() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status.Executing = true
	t.history.LastStart = time.Now()
}

func (t *Task) reportStop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status.Executing = false
	t.history.LastStop = time.Now()
}

func (t *Task) restart(c *Config) bool {
	if c.Interval == 0 {
		return true
	}
	select {
	case <-t.stop:
		return false
	case <-t.skipWait:
	case <-time.After(time.Second * time.Duration(c.Interval)):
	}
	return true
}

func (t *Task) run(c *Config) bool {
	// Create the command
	cmd, err := c.ToCommand()
	if err != nil {
		t.reportError(err)
		return true
	}
	// Start the command
	if err := cmd.Start(); err != nil {
		t.reportError(err)
		return true
	}
	t.reportStart()

	// Wait for the command to stop
	ch := make(chan struct{})
	go func() {
		cmd.Wait()
		t.reportStop()
		close(ch)
	}()
	select {
	case <-ch:
		return true
	case <-t.stop:
		cmd.Process.Kill()
		<-ch
		return false
	}
}

func (t *Task) stopped() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}
