package transport

import (
	"io"
	"os"

	"github.com/progrium/qtalk-go/mux"
)

func DialIO(out io.WriteCloser, in io.ReadCloser) (*mux.Session, error) {
	return mux.New(&ioduplex{out, in}), nil
}

func DialStdio() (*mux.Session, error) {
	return DialIO(os.Stdout, os.Stdin)
}
