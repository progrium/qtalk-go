package talk

import (
	"context"
	"io"
	"testing"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/rpc"
)

func TestPeerBidirectional(t *testing.T) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := mux.DialIO(aw, ar)
	sessB, _ := mux.DialIO(bw, br)

	peerA := NewPeer(sessA, codec.JSONCodec{})
	peerB := NewPeer(sessB, codec.JSONCodec{})
	defer peerA.Close()
	defer peerB.Close()

	peerA.Handle("hello", rpc.HandlerFunc(func(r rpc.Responder, c *rpc.Call) {
		r.Return("A")
	}))
	peerB.Handle("hello", rpc.HandlerFunc(func(r rpc.Responder, c *rpc.Call) {
		r.Return("B")
	}))

	go peerA.Respond()
	go peerB.Respond()

	var retB string
	_, err := peerA.Call(context.Background(), "hello", nil, &retB)
	if err != nil {
		t.Fatal(err)
	}
	if retB != "B" {
		t.Fatal("unexpected return:", retB)
	}

	var retA string
	_, err = peerB.Call(context.Background(), "hello", nil, &retA)
	if err != nil {
		t.Fatal(err)
	}
	if retA != "A" {
		t.Fatal("unexpected return:", retA)
	}
}
