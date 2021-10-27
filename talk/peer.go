package talk

import (
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/rpc"
)

// Peer is a mux session, RPC client and responder, all in one.
type Peer struct {
	*mux.Session
	*rpc.Client
	*rpc.RespondMux
	codec.Codec
}

// NewPeer returns a Peer based on a session and codec.
func NewPeer(session *mux.Session, codec codec.Codec) *Peer {
	return &Peer{
		Session:    session,
		Codec:      codec,
		Client:     rpc.NewClient(session, codec),
		RespondMux: rpc.NewRespondMux(),
	}
}

// Close will close the underlying session.
func (p *Peer) Close() error {
	return p.Client.Close()
}

// Respond lets the Peer respond to incoming channels like
// a server, using any registered handlers.
func (p *Peer) Respond() {
	srv := &rpc.Server{Handler: p.RespondMux, Codec: p.Codec}
	srv.Respond(p.Session, nil)
}
