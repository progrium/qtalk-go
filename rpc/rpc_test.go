package rpc

import (
	"context"
	"io"
	"testing"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/transport"
)

func newPair(handler Handler, codec codec.Codec) (*Client, *Server) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := transport.DialIO(aw, ar)
	sessB, _ := transport.DialIO(bw, br)

	srv := &Server{
		Codec:   codec,
		Handler: handler,
	}
	go srv.Respond(sessA)

	return NewClient(sessB, codec), srv
}

func TestRPC(t *testing.T) {
	client, _ := newPair(HandlerFunc(func(r Responder, c *Call) {
		var in string
		if err := c.Receive(&in); err != nil {
			t.Fatal(err)
		}
		r.Return(in)
	}), codec.JSONCodec{})

	var out string
	resp, err := client.Call(context.Background(), "", "Hello world", &out)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Continue {
		t.Fatal("unexpected continue")
	}
	if out != "Hello world" {
		t.Fatalf("unexpected return: %#v", out)
	}

}
