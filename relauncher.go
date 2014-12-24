package executor

import (
	"sync"
	"time"
)

// History records a brief history of a Relauncher.
type History struct {
	LastStart time.Time
	LastStop  time.Time
	LastError time.Time
	Error     error
}

// Status stores the status of a Relauncher.
type Status int

const (
	STATUS_RUNNING    = iota
	STATUS_RESTARTING = iota
	STATUS_STOPPED    = iota
)

// Relauncher automatically relaunches a task at a regular interval.
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
	res := &Relauncher{sync.RWMutex{}, History{}, STATUS_RUNNING, interval, job,
		make(chan struct{}), make(chan struct{}, 1), make(chan struct{})}
	go res.loop()
	return res
}

// History returns the Relauncher's history.
func (t *Relauncher) History() History {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.history
}

// HistoryStatus returns both the Relauncher's history and it's status.
func (t *Relauncher) HistoryStatus() (History, Status) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.history, t.status
}

// Interval returns the relaunch interval for a given relauncher.
func (t *Relauncher) Interval() time.Duration {
	return t.interval
}

// SkipWait skips the current relaunch timeout if applicable.
// If the job was not relaunching, this returns ErrNotWaiting.
func (t *Relauncher) SkipWait() error {
	select {
	case t.skipWait <- struct{}{}:
		return nil
	default:
		return ErrNotWaiting
	}
}

// Status returns the Relauncher's status
func (t *Relauncher) Status() Status {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.status
}

// Stop stops the relauncher.
// This method waits for the background job to terminate if necessary.
func (t *Relauncher) Stop() {
	// Send async stop message
	select {
	case t.stop <- struct{}{}:
	default:
	}
	<-t.onDone
}

// Wait waits for the relauncher to be stopped.
func (t *Relauncher) Wait() {
	<-t.onDone
}

func (t *Relauncher) loop() {
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

func (t *Relauncher) reportError(err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.history.LastError = time.Now()
	t.history.Error = err
}

func (t *Relauncher) reportStart() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = STATUS_RUNNING
	t.history.LastStart = time.Now()
}

func (t *Relauncher) reportStop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.status = STATUS_RESTARTING
	t.history.LastStop = time.Now()
}

func (t *Relauncher) restart() bool {
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

func (t *Relauncher) run() bool {
	// Start the job
	if t.job.Start() != nil {
		return true
	}
	t.reportStart()

	// Wait for the command to stop
	ch := make(chan struct{})
	go func() {
		if err := t.job.Wait(); err != nil {
			t.reportError(err)
		}
		t.reportStop()
		close(ch)
	}()

	// Wait for the stop signal or the done signal
	select {
	case <-ch:
		return true
	case <-t.stop:
		t.job.Stop()
		<-ch
		return false
	}
}

func (t *Relauncher) stopped() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}
