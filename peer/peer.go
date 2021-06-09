package peer

import (
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport"
)

type Peer struct {
	transport.Session
	codec.Codec
	*rpc.Client
	*rpc.RespondMux
}

func NewPeer(session transport.Session, codec codec.Codec) *Peer {
	return &Peer{
		Session:    session,
		Codec:      codec,
		Client:     rpc.NewClient(session, codec),
		RespondMux: rpc.NewRespondMux(),
	}
}

func (p *Peer) Respond() {
	srv := &rpc.Server{Handler: p.RespondMux}
	srv.Respond(p.Session)
}
