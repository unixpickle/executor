package executor

import (
	"os/exec"
	"sync"
)

type Task struct {
	mutex   sync.Mutex
	config  *TaskConfig
	stop    chan struct{}
	command *exec.Cmd
}

func NewTask(config *TaskConfig) *Task {
	return &Task{sync.Mutex{}, config.Clone(), nil, nil}
}

func (t *Task) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if stopChan != nil {
		return ErrAlreadyRunning
	}
	t.stop = make(chan struct{})
	t.command = t.config.ToCommand()
	if err := t.command.Start(); err != nil {
		t.stop = nil
		t.command = nil
		return err
	}
	go t.runCommand(t.command, t.stop)
}

func (t *Task) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.stop == nil {
		return ErrNotRunning
	}
	t.command.Process.Kill()
	<-t.stop
	return nil
}

func (t *Task) Wait() error {
	if ch := WaitChannel(); ch != nil {
		<-ch
		return nil
	} else {
		return ErrNotRunning
	}
}

func (t *Task) WaitChannel() chan struct{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.stop
}

func (t *Task) runCommand(command *exec.Cmd, stop chan struct{}) {
	command.Wait()
	close(stop)
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.command == command {
		t.command = nil
		t.stop = nil
	}
}

