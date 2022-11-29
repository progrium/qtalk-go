package mux

import (
	"io"
	"os"
)

// DialIO establishes a mux session using a WriterCloser and ReadCloser.
func DialIO(out io.WriteCloser, in io.ReadCloser) (Session, error) {
	return New(&ioduplex{out, in}), nil
}

// DialIO establishes a mux session using Stdout and Stdin.
func DialStdio() (Session, error) {
	return DialIO(os.Stdout, os.Stdin)
}
