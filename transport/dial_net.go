package transport

import (
	"net"

	"github.com/progrium/qtalk-go/mux"
)

func dialNet(proto, addr string) (*mux.Session, error) {
	conn, err := net.Dial(proto, addr)
	if err != nil {
		return nil, err
	}
	return mux.New(conn), nil
}

// DialTCP establishes a mux session via TCP connection.
func DialTCP(addr string) (*mux.Session, error) {
	return dialNet("tcp", addr)
}

// DialUnix establishes a mux session via Unix domain socket.
func DialUnix(path string) (*mux.Session, error) {
	return dialNet("unix", path)
}
