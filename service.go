package executor

import (
	"sync"
	"time"
)

// Service is a restartable Job which runs in the background.
type Service interface {
	// History returns a brief history of the Service.
	History() History

	// HistoryStatus returns the history and the status of a service.
	HistoryStatus() (History, Status)

	// SkipWait skips a wait if the service is a Relauncher and is waiting.
	SkipWait() error

	// Start starts the service if it is not already running.
	// By the time this returns, Status() must not be STATUS_STOPPED unless an
	// error is returned.
	Start() error

	// Status returns the status of the service.
	Status() Status

	// Stop stops the service synchronously.
	// By the time this returns, Status() must be STATUS_STOPPED unless an error
	// is returned.
	Stop() error
	
	// Wait waits for the service to stop.
	// Like Stop(), Wait() will only return once Status() has been set to
	// STATUS_STOPPED.
	Wait() error
}

// RelaunchService creates a service that relaunches a specific Job.
func RelaunchService(job Job, interval time.Duration) Service {
	res := new(relauncherService)
	res.job = job
	res.interval = interval
	return res
}

// JobService creates a Service that runs a specified Job.
func JobService(job Job) Service {
	res := new(jobService)
	res.job = job
	return res
}

type jobService struct {
	job     Job
	history History
	onDone  chan struct{}
	mutex   sync.Mutex
}

func (j *jobService) History() History {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	return j.history
}

func (j *jobService) HistoryStatus() (History, Status) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.onDone != nil {
		return j.history, STATUS_RUNNING
	}
	return j.history, STATUS_STOPPED
}

func (j *jobService) SkipWait() error {
	return ErrNotWaiting
}

func (j *jobService) Start() error {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.onDone != nil {
		return ErrAlreadyRunning
	}

	// We need to start the job while the service is still locked, that way
	// nobody stops the job before it *really* started.
	if err := j.job.Start(); err != nil {
		j.history.LastError = time.Now()
		j.history.Error = err
		return err
	}

	// Setup the history and other fields
	j.history.LastStart = time.Now()
	j.onDone = make(chan struct{})

	// In the background, wait for the job to finish.
	go func() {
		err := j.job.Wait()
		j.mutex.Lock()
		defer j.mutex.Unlock()
		j.history.LastStop = time.Now()
		if err != nil {
			j.history.LastError = time.Now()
			j.history.Error = err
		}
		close(j.onDone)
		j.onDone = nil
	}()

	return nil
}

func (j *jobService) Status() Status {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	if j.onDone != nil {
		return STATUS_RUNNING
	} else {
		return STATUS_STOPPED
	}
}

func (j *jobService) Stop() error {
	j.mutex.Lock()
	if j.onDone == nil {
		j.mutex.Unlock()
		return ErrNotRunning
	}

	// Stop the job and wait for the background Goroutine to finish
	j.job.Stop()
	ch := j.onDone
	j.mutex.Unlock()
	<-ch

	return nil
}

func (j *jobService) Wait() error {
	j.mutex.Lock()
	ch := j.onDone
	j.mutex.Unlock()
	if ch == nil {
		return ErrNotRunning
	}
	<-ch
	return nil
}

type relauncherService struct {
	job         Job
	interval    time.Duration
	mutex       sync.Mutex
	relauncher  *Relauncher
	lastHistory History
}

func (r *relauncherService) History() History {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.relauncher == nil {
		return r.lastHistory
	}
	return r.relauncher.History()
}

func (r *relauncherService) HistoryStatus() (History, Status) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.relauncher == nil {
		return r.lastHistory, STATUS_STOPPED
	}
	return r.HistoryStatus()
}

func (r *relauncherService) SkipWait() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.relauncher == nil {
		return ErrNotRunning
	}
	return r.SkipWait()
}

func (r *relauncherService) Start() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.relauncher != nil {
		return ErrAlreadyRunning
	}
	r.relauncher = Relaunch(r.job, r.interval)
	return nil
}

func (r *relauncherService) Status() Status {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.relauncher == nil {
		return STATUS_STOPPED
	}
	return r.relauncher.Status()
}

func (r *relauncherService) Stop() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.relauncher == nil {
		return ErrNotRunning
	}
	r.relauncher.Stop()
	r.lastHistory = r.relauncher.History()
	r.relauncher = nil
	return nil
}

func (r *relauncherService) Wait() error {
	// Even though r.relauncher.Wait() may return before the corresponding
	// Stop() call sets lastHistory, any calls to History() will seize the mutex
	// and thus wait for the Stop() call to finish anyway.
	r.mutex.Lock()
	x := r.relauncher
	r.mutex.Unlock()
	if x == nil {
		return ErrNotRunning
	}
	x.Wait()
	return nil
}
