package rpctest

import (
	"io"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport/qmux"
)

func NewPair(handler rpc.Handler, codec codec.Codec) (*rpc.Client, *rpc.Server) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := qmux.DialIO(aw, ar)
	sessB, _ := qmux.DialIO(bw, br)

	srv := &rpc.Server{
		Codec:   codec,
		Handler: handler,
	}
	go srv.Respond(sessA)

	return rpc.NewClient(sessB, codec), srv
}
