package executor

import (
	"os/exec"
	"sync"
	"time"
)

type Status struct {
	Running   bool
	Looping   bool
	LastStart time.Time
	LastStop  time.Time
	LastError string
}

type Job struct {
	mutex  sync.RWMutex
	config *Config
	wait   chan struct{}
	kill   chan struct{}
}

func NewJob(config *Config) *Job {
	return &Job{sync.RWMutex{}, config.Clone(), nil, nil}
}

func (j *Job) Start() error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.wait != nil {
		return ErrAlreadyRunning
	}
	j.wait = make(chan struct{})
	j.kill = make(chan struct{})
	go j.runLoop(j.wait, j.kill)
	return nil
}

func (j *Job) Stop() error {
	
}

func (j *Job) runLoop(wait chan struct{}, kill chan struct{}) {
	for {
		
	}
}

