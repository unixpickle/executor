package executor

import (
	"os/exec"
	"sync/atomic"
)

type Status struct {
	Running   bool
	LastStart time.Time
	LastStop  time.Time
	LastError time.Time
	Error     string
}

type Task struct {
	status   atomic.Value
	mutex    sync.Mutex
	onDone   chan struct{}
	stop     chan struct{}
	skipWait chan struct{}
}

func StartTask(config *Config, lastStatus Status) *Task {
	res := &Task{atomic.Value{}, sync.Mutex{}, make(chan struct{}),
		make(chan struct{}, 1), make(chan struct{})}
	res.status.Store(lastStatus)
	go res.loop(config.Clone())
	return res
}

func (t *Task) Done() bool {
	select {
	case <-t.onDone:
		return true
	default:
		return false
	}
}

func (t *Task) SkipWait() {
	select {
	case t.skipWait <- struct{}{}:
	default:
	}
}

func (t *Task) Status() Status {
	return t.status.Load().(Status)
}

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
	s := t.Status()
	s.LastError = time.Now()
	s.Error = err.Error()
	t.setStatus(s)
}

func (t *Task) reportStart() {
	s := t.Status()
	s.LastStart = time.Now()
	t.setStatus(s)
}

func (t *Task) reportStop() {
	s := t.Status()
	s.LastStop = time.Now()
	t.setStatus(s)
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

func (t *Task) setStatus(s Status) {
	t.status.Store(s)
}
