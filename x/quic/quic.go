package quic

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/talk"
)

func New(conn quic.Connection) mux.Session {
	return &session{conn}
}

var defaultTLSConfig = tls.Config{
	NextProtos: []string{"qtalk-quic"},
}

func Dial(addr string) (mux.Session, error) {
	conn, err := quic.DialAddr(addr, &defaultTLSConfig, nil)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

func init() {
	talk.Dialers["quic"] = Dial
}

type session struct {
	conn quic.Connection
}

func (s *session) Close() error {
	return s.conn.CloseWithError(42, "close connection")
}

func (s *session) Accept() (mux.Channel, error) {
	stream, err := s.conn.AcceptStream(context.Background())
	if err != nil {
		return nil, err
	}
	header := make([]byte, 1)
	_, err = stream.Read(header)
	if err != nil {
		return nil, err
	}
	return &channel{stream}, nil
}

func (s *session) Open(ctx context.Context) (mux.Channel, error) {
	// TODO Make this wait for an acknowledgement from the remote that it has
	// accepted the connection. It writes some data in order to notify the remote
	// of the new stream immediately, but my initial attempt to send an
	// acknowledgement from the remote side lead to deadlocks in the tests.
	stream, err := s.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	_, err = stream.Write([]byte("!"))
	if err != nil {
		return nil, err
	}
	return &channel{stream}, nil
}

func (s *session) Wait() error {
	<-s.conn.Context().Done()
	return s.conn.Context().Err()
}

type channel struct {
	stream quic.Stream
}

func (c *channel) ID() uint32 {
	return uint32(c.stream.StreamID())
}

func (c *channel) Read(p []byte) (int, error) {
	return c.stream.Read(p)
}

func (c *channel) Write(p []byte) (int, error) {
	return c.stream.Write(p)
}

func (c *channel) Close() error {
	c.stream.CancelRead(42)
	return c.CloseWrite()
}

func (c *channel) CloseWrite() error {
	// TODO this may need a lock to avoid concurrent call with Write
	return c.stream.Close()
}
