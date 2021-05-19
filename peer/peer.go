package rpc

import (
	"context"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport"
)

type Peer struct {
	transport.Session

	caller    rpc.Caller
	responder *rpc.ServeMux
}

func NewPeer(session transport.Session, codec codec.Codec) *Peer {
	return &Peer{
		Session:   session,
		caller:    rpc.NewClient(session, codec),
		responder: rpc.NewServeMux(codec),
	}
}

func (p *Peer) Respond() {
	srv := &rpc.Server{Mux: p.responder}
	srv.Respond(p.Session)
}

func (p *Peer) Call(ctx context.Context, selector string, args, reply interface{}) (*rpc.Response, error) {
	return p.caller.Call(ctx, selector, args, reply)
}

func (p *Peer) Bind(path string, v interface{}) {
	p.responder.Bind(path, v)
}
