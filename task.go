package executor

import (
	"os/exec"
	"sync/atomic"
)

// History records the history of a Task including its last launch,
// exit, and error.
type History struct {
	Running   bool
	LastStart time.Time
	LastStop  time.Time
	LastError time.Time
	Error     string
}

// Task stores runtime information for a task.
type Task struct {
	history  atomic.Value
	mutex    sync.Mutex
	onDone   chan struct{}
	stop     chan struct{}
	skipWait chan struct{}
}

// StartTask begins the execution loop for a given task.
func StartTask(config *Config) *Task {
	res := &Task{atomic.Value{}, sync.Mutex{}, make(chan struct{}),
		make(chan struct{}, 1), make(chan struct{})}
	res.setHistory(History{})
	go res.loop(config.Clone())
	return res
}

// Done returns true if a task has been stopped or exited and was not set to
// relaunch.
func (t *Task) Done() bool {
	select {
	case <-t.onDone:
		return true
	default:
		return false
	}
}

// History returns the task's history.
func (t *Task) History() History {
	return t.history.Load().(History)
}

// SkipWait skips the relaunch timeout if it is actively running.
func (t *Task) SkipWait() {
	select {
	case t.skipWait <- struct{}{}:
	default:
	}
}

// Stop stops he the task.
// This method waits for the background process and goroutines to terminate
// before returning.
func (t *Task) Stop() {
	// Send async stop message
	select {
	case t.stop <- struct{}{}:
	default:
	}
	<-t.onDone
}

func (t *Task) loop(c *Config) {
	for !t.Done() {
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
	close(t.onDone)
}

func (t *Task) reportError(err error) {
	s := t.History()
	s.LastError = time.Now()
	s.Error = err.Error()
	t.setHistory(s)
}

func (t *Task) reportStart() {
	s := t.History()
	s.LastStart = time.Now()
	s.Running = true
	t.setHistory(s)
}

func (t *Task) reportStop() {
	s := t.History()
	s.LastStop = time.Now()
	s.Running = false
	t.setHistory(s)
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

func (t *Task) setHistory(h History) {
	t.history.Store(h)
}
