package peer

import (
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/rpc"
)

type Peer struct {
	*mux.Session
	*rpc.Client
	*rpc.RespondMux
	codec.Codec
}

func New(session *mux.Session, codec codec.Codec) *Peer {
	return &Peer{
		Session:    session,
		Codec:      codec,
		Client:     rpc.NewClient(session, codec),
		RespondMux: rpc.NewRespondMux(),
	}
}

func (p *Peer) Respond() {
	srv := &rpc.Server{Handler: p.RespondMux, Codec: p.Codec}
	srv.Respond(p.Session)
}
