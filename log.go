package executor

import "io"

// Log is a general log destination for a command's output.
type Log interface {
	Open() (io.WriteCloser, error)
}

// NullLog is a Log which returns no-op writers.
var NullLog = nullLog{}

type nopWriteCloser struct{}

func (c nopWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (c nopWriteCloser) Close() error {
	return nil
}

type nullLog struct{}

func (x nullLog) Open() (io.WriteCloser, error) {
	return nopWriteCloser{}, nil
}

