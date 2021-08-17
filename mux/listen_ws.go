package mux

import (
	"io"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

// WSListener wraps a net.Listener and WebSocket server to return connected mux sessions.
type WSListener struct {
	net.Listener
	accepted chan *Session
}

// Accept waits for and returns the next connected session to the listener.
func (l *WSListener) Accept() (*Session, error) {
	sess, ok := <-l.accepted
	if !ok {
		return nil, io.EOF
	}
	return sess, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *WSListener) Close() error {
	close(l.accepted)
	return l.Listener.Close()
}

// HandleWS is used to take WebSocket connections, wrap as mux sessions, and send to a NetListener to be accepted.
func HandleWS(l *WSListener, ws *websocket.Conn) {
	ws.PayloadType = websocket.BinaryFrame
	sess := New(ws)
	defer sess.Close()
	l.accepted <- sess
	sess.Wait()
}

// ListenWS takes a TCP address and returns a NetListener with an HTTP+WebSocket server listening on the given address.
func ListenWS(addr string) (*WSListener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	wsl := &WSListener{
		Listener: l,
		accepted: make(chan *Session),
	}
	srv := &http.Server{
		Addr: addr,
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			HandleWS(wsl, ws)
		}),
	}
	go srv.Serve(l)
	return wsl, nil
}
