package mux

import (
	"net"
)

// netListener wraps a net.Listener to return connected mux sessions.
type netListener struct {
	net.Listener
}

// Accept waits for and returns the next connected session to the listener.
func (l *netListener) Accept() (Session, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *netListener) Close() error {
	return l.Listener.Close()
}

func (l *netListener) Addr() net.Addr {
	return l.Listener.Addr()
}

func ListenerFrom(l net.Listener) Listener {
	return &netListener{Listener: l}
}

// ListenTCP creates a TCP listener at the given address.
func ListenTCP(addr string) (Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return ListenerFrom(l), nil
}

// ListenTCP creates a Unix domain socket listener at the given path.
func ListenUnix(path string) (Listener, error) {
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	return ListenerFrom(l), nil
}
