package transport

import (
	"net"
	"net/http"

	"github.com/progrium/qtalk-go/mux"
	"golang.org/x/net/websocket"
)

// HandleWS is used to take WebSocket connections, wrap as mux sessions, and send to a NetListener to be accepted.
func HandleWS(l *NetListener, ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	sess := mux.New(ws)
	defer sess.Close()
	l.accepted <- sess
	l.errs <- sess.Wait()
}

// ListenWS takes a TCP address and returns a NetListener with an HTTP+WebSocket server listening on the given address.
func ListenWS(addr string) (*NetListener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	nl := &NetListener{
		Listener: l,
		accepted: make(chan *mux.Session),
		errs:     make(chan error, 2),
		closer:   make(chan bool, 1),
	}
	s := &http.Server{
		Addr: addr,
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			HandleWS(nl, ws)
		}),
	}
	go func() {
		nl.errs <- s.Serve(l)
	}()
	return nl, nil
}
