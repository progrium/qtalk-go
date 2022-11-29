package mux

import (
	"fmt"

	"golang.org/x/net/websocket"
)

// DialWS establishes a mux session via WebSocket connection.
// The address must be a host and port. Opening a WebSocket
// connection at a particular path is not supported.
func DialWS(addr string) (Session, error) {
	ws, err := websocket.Dial(fmt.Sprintf("ws://%s/", addr), "", fmt.Sprintf("http://%s/", addr))
	if err != nil {
		return nil, err
	}
	ws.PayloadType = websocket.BinaryFrame
	return New(ws), nil
}
