package transport

import (
	"io"
	"os"

	"github.com/progrium/qtalk-go/mux"
)

// DialIO establishes a mux session using a WriterCloser and ReadCloser.
func DialIO(out io.WriteCloser, in io.ReadCloser) (*mux.Session, error) {
	return mux.New(&ioduplex{out, in}), nil
}

// DialIO establishes a mux session using Stdout and Stdin.
func DialStdio() (*mux.Session, error) {
	return DialIO(os.Stdout, os.Stdin)
}
