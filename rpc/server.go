package rpc

import (
	"io"
	"log"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/transport"
)

type Server struct {
	Handler Handler
	Codec   codec.Codec
}

func (s *Server) Serve(l transport.Listener) error {
	for {
		sess, err := l.Accept()
		if err != nil {
			return err
		}
		go s.Respond(sess)
	}
}

func (s *Server) Respond(sess transport.Session) {
	for {
		ch, err := sess.Accept()
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err)
		}
		go s.respond(sess, ch)
	}
}

func (s *Server) respond(sess transport.Session, ch transport.Channel) {
	defer ch.Close()

	framer := &codec.FrameCodec{Codec: s.Codec}
	dec := framer.Decoder(ch)

	var call Call
	err := dec.Decode(&call)
	if err != nil {
		log.Println("rpc.Respond:", err)
		return
	}

	call.Decoder = dec
	call.Caller = &Client{
		session: sess,
		codec:   s.Codec,
	}

	header := &ResponseHeader{}
	resp := &responder{
		ch:     ch,
		c:      framer,
		header: header,
	}

	if s.Handler == nil {
		s.Handler = NewRespondMux()
	}
	s.Handler.RespondRPC(resp, &call)
}
