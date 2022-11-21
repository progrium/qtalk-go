package rpc

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
)

// Server wraps a Handler and codec to respond to RPC calls.
type Server struct {
	Handler Handler
	Codec   codec.Codec
}

// ServeMux will Accept sessions until the Listener is closed, and will Respond to accepted sessions in their own goroutine.
func (s *Server) ServeMux(l mux.Listener) error {
	for {
		sess, err := l.Accept()
		if err != nil {
			return err
		}
		go s.Respond(sess, nil)
	}
}

// Serve will Accept sessions until the Listener is closed, and will Respond to accepted sessions in their own goroutine.
func (s *Server) Serve(l net.Listener) error {
	return s.ServeMux(mux.ListenerFrom(l))
}

// Respond will Accept channels until the Session is closed and respond with the server handler in its own goroutine.
// If Handler was not set, an empty RespondMux is used. If the handler does not initiate a response, a nil value is
// returned. If the handler does not call Continue, the channel will be closed. Respond will panic if Codec is nil.
//
// If the context is not nil, it will be added to Calls. Otherwise the Call Context will be set to a context.Background().
func (s *Server) Respond(sess *mux.Session, ctx context.Context) {
	defer sess.Close()

	if s.Codec == nil {
		panic("rpc.Respond: nil codec")
	}

	hn := s.Handler
	if hn == nil {
		hn = NewRespondMux()
	}

	for {
		ch, err := sess.Accept()
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err)
		}
		go s.respond(hn, sess, ch, ctx)
	}
}

func (s *Server) respond(hn Handler, sess *mux.Session, ch *mux.Channel, ctx context.Context) {
	framer := &FrameCodec{Codec: s.Codec}
	dec := framer.Decoder(ch)

	var call Call
	err := dec.Decode(&call)
	if err != nil {
		log.Println("rpc.Respond:", err)
		return
	}

	call.Selector = cleanSelector(call.Selector)
	call.Decoder = dec
	call.Caller = &Client{
		Session: sess,
		codec:   s.Codec,
	}
	if ctx == nil {
		call.Context = context.Background()
	} else {
		call.Context = ctx
	}
	call.ch = ch

	header := &ResponseHeader{}
	resp := &responder{
		ch:     ch,
		c:      framer,
		header: header,
	}

	hn.RespondRPC(resp, &call)
	if !resp.responded {
		resp.Return()
	}
	if !resp.header.Continue {
		ch.Close()
	}
}
