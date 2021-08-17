package mux

import (
	"io"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

// wsListener wraps a net.Listener and WebSocket server to return connected mux sessions.
type wsListener struct {
	net.Listener
	accepted chan *Session
}

// Accept waits for and returns the next connected session to the listener.
func (l *wsListener) Accept() (*Session, error) {
	sess, ok := <-l.accepted
	if !ok {
		return nil, io.EOF
	}
	return sess, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *wsListener) Close() error {
	close(l.accepted)
	return l.Listener.Close()
}

func (l *wsListener) Addr() net.Addr {
	return l.Listener.Addr()
}

// ListenWS takes a TCP address and returns a Listener for a HTTP+WebSocket server listening on the given address.
func ListenWS(addr string) (Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	wsl := &wsListener{
		Listener: l,
		accepted: make(chan *Session),
	}
	srv := &http.Server{
		Addr: addr,
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			ws.PayloadType = websocket.BinaryFrame
			sess := New(ws)
			defer sess.Close()
			wsl.accepted <- sess
			sess.Wait()
		}),
	}
	go srv.Serve(l)
	return wsl, nil
}
