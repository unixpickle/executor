package executor

import (
	"github.com/unixpickle/spinlog"
	"io"
)

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

type nopWriteCloser struct {
}

func (c nopWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (c nopWriteCloser) Close() error {
	return nil
}
