package executor

import (
	"os/exec"
	"sync/atomic"
)

// History records the history of a Task including its last launch,
// exit, and error.
type History struct {
	Running   bool
	Done      bool
	LastStart time.Time
	LastStop  time.Time
	LastError time.Time
	Error     string
}

// Status records the running status of a task.
type Status struct {
	Executing bool
	Done      bool
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

func (t *Task) loop(c *Config) {
	for !t.stopped() {
		if !t.run() {
			break
		}
		if !c.Relaunch {
			break
		}
		if !t.restart() {
			break
		}
	}
	t.mutex.Lock()
	t.Status.Done = true
	t.mutex.Unlock()
	close(t.onDone)
}

func (t *Task) reportError(err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.History.LastError = time.Now()
	t.History.Error = err.Error()
}

func (t *Task) reportStart() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Status.Running = true
	t.History.LastStart = time.Now()
}

func (t *Task) reportStop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Status.Running = false
	t.History.LastStop = time.Now()
}

func (t *Task) restart() bool {
	select {
	case <-t.stop:
		return false
	case <-t.skipWait:
	case <-time.After(time.Second * t.Interval):
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
