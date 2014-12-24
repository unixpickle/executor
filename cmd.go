package executor

import (
	"io"
	"os/exec"
	"sync"
)

// Cmd is the full configuration for a command-line executable.
type Cmd struct {
	Stdout      Log
	Stderr      Log
	Directory   string
	SetUID      bool
	UID         int
	SetGID      bool
	GID         int
	Arguments   []string
	Environment map[string]string
}

// Command creates a new Cmd with generic settings given a set of command-line
// arguments.
func Command(arguments ...string) *Cmd {
	res := new(Cmd)
	res.Arguments = arguments
	res.Stdout = NullLog
	res.Stderr = NullLog
	return res
}

// Clone creates a copy of a Cmd.
// While it does do a completey copy of the Arguments and Environment fields, it
// cannot copy the Logs.
func (c *Cmd) Clone() *Cmd {
	x := *c
	cpy := &x
	cpy.Arguments = make([]string, len(c.Arguments))
	for i, arg := range c.Arguments {
		cpy.Arguments[i] = arg
	}
	cpy.Environment = map[string]string{}
	for key, val := range c.Environment {
		cpy.Environment[key] = val
	}
	return cpy
}

// ToJob creates a Job based on the current configuration in a Cmd.
// If the receiver is modified after a call to ToJob(), the job will not be
// modified.
func (c *Cmd) ToJob() Job {
	return &cmdJob{sync.Mutex{}, c.Clone(), nil}
}

func (c *Cmd) toExecCmd() (*exec.Cmd, error) {
	task := exec.Command(c.Arguments[0], c.Arguments[1:]...)
	for key, value := range c.Environment {
		task.Env = append(task.Env, key+"="+value)
	}

	// TODO: here, set UID and GID

	task.Dir = c.Directory

	// Create output streams
	var err error
	if task.Stdout, err = c.Stdout.Open(); err != nil {
		return nil, err
	}
	if task.Stderr, err = c.Stderr.Open(); err != nil {
		task.Stdout.(io.Closer).Close()
		return nil, err
	}

	return task, nil
}

type cmdJob struct {
	mutex   sync.Mutex
	command *Cmd
	execCmd *exec.Cmd
}

func (c *cmdJob) Start() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Make sure the job is not already running.
	if c.execCmd != nil {
		return ErrAlreadyRunning
	}

	// Generate the exec.Cmd
	cmd, err := c.command.toExecCmd()
	if err != nil {
		return err
	}

	// Start the command or return an error
	if err := cmd.Start(); err != nil {
		cmd.Stdout.(io.Closer).Close()
		cmd.Stderr.(io.Closer).Close()
		return err
	}

	c.execCmd = cmd
	return nil
}

func (c *cmdJob) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.execCmd == nil {
		return ErrNotRunning
	}
	c.execCmd.Process.Kill()
	return nil
}

func (c *cmdJob) Wait() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.execCmd == nil {
		return ErrNotRunning
	}

	c.mutex.Unlock()
	res := c.execCmd.Wait()
	c.mutex.Lock()

	c.execCmd.Stdout.(io.Closer).Close()
	c.execCmd.Stderr.(io.Closer).Close()
	c.execCmd = nil
	return res
}
