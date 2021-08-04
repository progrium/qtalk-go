package rpctest

import (
	"io"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport"
)

// NewPair creates a Client and Server connected by in-memory pipes.
// The server Respond method is called in a goroutine. Only the client
// should need to be cleaned up with call to Close.
func NewPair(handler rpc.Handler, codec codec.Codec) (*rpc.Client, *rpc.Server) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := transport.DialIO(aw, ar)
	sessB, _ := transport.DialIO(bw, br)

	srv := &rpc.Server{
		Codec:   codec,
		Handler: handler,
	}
	go srv.Respond(sessA)

	return rpc.NewClient(sessB, codec), srv
}
