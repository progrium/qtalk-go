package transport

import (
	"fmt"

	"github.com/progrium/qtalk-go/mux"
	"golang.org/x/net/websocket"
)

func DialWS(addr string) (*mux.Session, error) {
	ws, err := websocket.Dial(fmt.Sprintf("ws://%s/", addr), "", fmt.Sprintf("http://%s/", addr))
	if err != nil {
		return nil, err
	}
	ws.PayloadType = websocket.BinaryFrame
	return mux.New(ws), nil
}
