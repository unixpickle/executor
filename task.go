package executor

import (
	"os/exec"
	"sync"
)

type Task struct {
	mutex    sync.Mutex
	config   *TaskConfig
	stopChan chan struct{}
	cmd      *exec.Cmd
}

func NewTask(config *TaskConfig) *Task {
	return &Task{sync.Mutex{}, config.Clone(), nil}
}

func (t *Task) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if stopChan != nil {
		return ErrAlreadyRunning
	}
	t.stopChan = make(chan struct{})
}

