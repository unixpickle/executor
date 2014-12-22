package executor


type Job struct {
}

func NewJob(config *Config) *Job {
	return nil
}

func (j *Job) Start() error {
	return nil
}

func (j *Job) Status() (bool, Status) {
	return false, Status{}
}

func (j *Job) Stop() error {
	j.mutex.Lock()
	if j.wait == nil {
		j.mutex.Unlock()
		return ErrNotRunning
	}
	j.wait.Stop()
	j.wait = nil
	ch := j.stopped
	j.stopped = nil
	j.mutex.Unlock()
	<-ch
	return nil
}

type struct bgContext {
	job     *Job
	wait    *golocks.WaitMutex
	stopped chan struct{}
}

