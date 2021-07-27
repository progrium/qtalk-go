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

func DialTCP(addr string) (*mux.Session, error) {
	return dialNet("tcp", addr)
}

func DialUnix(addr string) (*mux.Session, error) {
	return dialNet("unix", addr)
}
