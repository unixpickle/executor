package executor

import (
	"sync"
	"time"
)

// History records a brief history of a Relauncher.
type History struct {
	LastStart time.Time `json:"last_start"`
	LastStop  time.Time `json:"last_stop"`
	LastError time.Time `json:"last_error"`
	Error     string    `json:"error"`
}

// Status stores the status of a Relauncher.
type Status int

const (
	STATUS_RUNNING    = iota
	STATUS_RESTARTING = iota
	STATUS_STOPPED    = iota
)

// Task stores runtime information for a task.
type Relauncher struct {
	mutex   sync.RWMutex
	history History
	status  Status

	interval time.Duration
	job      Job
	
	onDone   chan struct{}
	stop     chan struct{}
	skipWait chan struct{}
}

// Relaunch starts a job and continually relaunches it on a given interval.
func Relaunch(job Job, interval time.Duration) *Relauncher {
	res := &Task{sync.RWMutex{}, History{}, Status{}, interval, job,
		make(chan struct{}), make(chan struct{}, 1), make(chan struct{})}
	go res.loop()
	return res
}

// History returns the Relauncher's history.
func (t *Task) History() History {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.history
}

// HistoryStatus returns both the Relauncher's history and it's status.
func (t *Task) HistoryStatus() (History, Status) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.history, t.status
}

// SkipWait skips the current relaunch timeout if applicable.
// If the job was not relaunching, this returns ErrNotWaiting.
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

// Stop stops the relauncher.
// This method waits for the background job to terminate if necessary.
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

func (t *Task) loop(j Job) {
	for !t.stopped() {
		if !t.run() {
			break
		}
		if !t.restart() {
			break
		}
	}
	t.mutex.Lock()
	t.status = STATUS_STOPPED
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
	t.status = STATUS_RUNNING
	t.history.LastStart = time.Now()
}

func (t *Task) reportStop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = STATUS_RESTARTING
	t.history.LastStop = time.Now()
}

func (t *Task) restart() bool {
	if t.interval == 0 {
		return true
	}
	select {
	case <-t.stop:
		return false
	case <-t.skipWait:
	case <-time.After(t.interval):
	}
	return true
}

func (t *Task) run() bool {
	// Wait for the command to stop
	ch := make(chan struct{})
	go func() {
		t.job.Run()
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
