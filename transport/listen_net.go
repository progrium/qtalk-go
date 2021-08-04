package transport

import (
	"io"
	"net"

	"github.com/progrium/qtalk-go/mux"
)

// NetListener wraps a net.Listener to return connected mux sessions.
type NetListener struct {
	net.Listener
	accepted chan *mux.Session
	closer   chan bool
	errs     chan error
}

// Accept waits for and returns the next connected session to the listener.
func (l *NetListener) Accept() (*mux.Session, error) {
	select {
	case <-l.closer:
		return nil, io.EOF
	case err := <-l.errs:
		return nil, err
	case sess := <-l.accepted:
		return sess, nil
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *NetListener) Close() error {
	if l.closer != nil {
		l.closer <- true
	}
	return l.Listener.Close()
}

func listenNet(proto, addr string) (*NetListener, error) {
	l, err := net.Listen(proto, addr)
	if err != nil {
		return nil, err
	}
	closer := make(chan bool, 1)
	errs := make(chan error, 1)
	accepted := make(chan *mux.Session)
	go func(l net.Listener) {
		for {
			conn, err := l.Accept()
			if err != nil {
				errs <- err
				return
			}
			accepted <- mux.New(conn)
		}
	}(l)
	return &NetListener{
		Listener: l,
		errs:     errs,
		accepted: accepted,
		closer:   closer,
	}, nil
}

// ListenTCP creates a TCP listener at the given address.
func ListenTCP(addr string) (*NetListener, error) {
	return listenNet("tcp", addr)
}

// ListenTCP creates a Unix domain socket listener at the given path.
func ListenUnix(path string) (*NetListener, error) {
	return listenNet("unix", path)
}
