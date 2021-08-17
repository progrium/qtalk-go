package mux

import (
	"net"
)

func dialNet(proto, addr string) (*Session, error) {
	conn, err := net.Dial(proto, addr)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// DialTCP establishes a mux session via TCP connection.
func DialTCP(addr string) (*Session, error) {
	return dialNet("tcp", addr)
}

// DialUnix establishes a mux session via Unix domain socket.
func DialUnix(path string) (*Session, error) {
	return dialNet("unix", path)
}
