package executor

import (
	"os/exec"
	"sync"
)

type Command struct {
	Stdout      Log               `json:"stdout"`
	Stderr      Log               `json:"stderr"`
	Directory   string            `json:"directory"`
	SetUID      bool              `json:"set_uid"`
	UID         int               `json:"uid"`
	SetGID      bool              `json:"set_gid"`
	GID         int               `json:"gid"`
	Arguments   []string          `json:"arguments"`
	Environment map[string]string `json:"environment"`
}

func (c *Command) Clone() *Command {
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

func (c *Command) ToCmd() (*exec.Cmd, error) {
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
		return nil, err
	}

	return task, nil
}

func (c *Command) ToJob() Job {
	return &cmdJob{sync.Mutex{}, c.Clone(), nil}
}

type cmdJob struct {
	mutex   sync.Mutex
	command *Command
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
	cmd, err := c.command.ToCmd()
	if err != nil {
		return err
	}

	// Start the command or return an error
	if err := cmd.Start(); err != nil {
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

	c.execCmd = nil
	return res
}
