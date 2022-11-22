package mux

import (
	"io"
	"net"
	"os"
)

// ioListener wraps a single ReadWriteCloser to use as a listener.
type ioListener struct {
	io.ReadWriteCloser
}

// Accept will always return the wrapped ReadWriteCloser as a mux session.
func (l *ioListener) Accept() (Session, error) {
	return New(l.ReadWriteCloser), nil
}

func (l *ioListener) Addr() net.Addr {
	return nil
}

type ioduplex struct {
	io.WriteCloser
	io.ReadCloser
}

func (d *ioduplex) Close() error {
	if err := d.WriteCloser.Close(); err != nil {
		return err
	}
	if err := d.ReadCloser.Close(); err != nil {
		return err
	}
	return nil
}

// ListenIO returns an IOListener that gives a mux session based on seperate
// WriteCloser and ReadClosers.
func ListenIO(out io.WriteCloser, in io.ReadCloser) (Listener, error) {
	return &ioListener{
		&ioduplex{out, in},
	}, nil
}

// ListenStdio is a convenience for calling ListenIO with Stdout and Stdin.
func ListenStdio() (Listener, error) {
	return ListenIO(os.Stdout, os.Stdin)
}
