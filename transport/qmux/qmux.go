package qmux

import (
	"context"
	"io"

	"github.com/progrium/qmux/golang/mux"
	qtransport "github.com/progrium/qmux/golang/transport"
	"github.com/progrium/qtalk-go/transport"
)

type Session struct {
	mux.Session
}

func (s *Session) Open(ctx context.Context) (transport.Channel, error) {
	return s.Session.Open(ctx)
}

func (s *Session) Accept() (transport.Channel, error) {
	return s.Session.Accept()
}

func DialIO(out io.WriteCloser, in io.ReadCloser) (transport.Session, error) {
	sess, err := qtransport.DialIO(out, in)
	return &Session{Session: sess}, err
}
