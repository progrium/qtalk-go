package mux

import (
	"net"
)

// NetListener wraps a net.Listener to return connected mux sessions.
type NetListener struct {
	net.Listener
}

// Accept waits for and returns the next connected session to the listener.
func (l *NetListener) Accept() (*Session, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *NetListener) Close() error {
	return l.Listener.Close()
}

func listenNet(proto, addr string) (*NetListener, error) {
	l, err := net.Listen(proto, addr)
	if err != nil {
		return nil, err
	}
	return &NetListener{Listener: l}, nil
}

// ListenTCP creates a TCP listener at the given address.
func ListenTCP(addr string) (*NetListener, error) {
	return listenNet("tcp", addr)
}

// ListenTCP creates a Unix domain socket listener at the given path.
func ListenUnix(path string) (*NetListener, error) {
	return listenNet("unix", path)
}
