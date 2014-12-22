package executor

import (
	"github.com/unixpickle/spinlog"
	"io"
	"io/ioutil"
	"os/exec"
)

type Identity struct {
	SetUID bool `json:"set_uid"`
	UID    int  `json:"uid"`
	SetGID bool `json:"set_gid"`
	GID    int  `json:"gid"`
}

type Log struct {
	Config spinlog.LineConfig `json:"config"`
	Enable bool               `json:"enable"`
	Lines  bool               `json:"lines"`
}

func (c *Log) Open() (io.WriteCloser, error) {
	if !c.Enable {
		return nopWriteCloser{}, nil
	} else if !c.Lines {
		return spinlog.NewLog(c.Config.Config)
	}
	return spinlog.NewLineLog(c.Config)
}

type Config struct {
	Stdout      Log               `json:"stdout"`
	Stderr      Log               `json:"stderr"`
	Directory   string            `json:"directory"`
	Identity    Identity          `json:"identity"`
	Arguments   []string          `json:"arguments"`
	Environment map[string]string `json:"environment"`
	Relaunch    bool              `json:"relaunch"`
	Interval    int               `json:"interval"`
}

func (c *Config) Clone() *Config {
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
}

func (c *Config) ToCommand() (*exec.Cmd, error) {
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

type nopWriteCloser struct{
}

func (c nopWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (c nopWriteCloser) Close() error {
	return nil
}

