package rpc

import (
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/transport"
)

type Responder interface {
	Header() *ResponseHeader
	Return(interface{}) error
	Continue(interface{}) (transport.Channel, error)
}

type responder struct {
	header *ResponseHeader
	ch     transport.Channel
	c      codec.Codec
}

func (r *responder) Header() *ResponseHeader {
	return r.header
}

func (r *responder) Return(v interface{}) error {
	if err := r.respond(v, false); err != nil {
		return err
	}
	return r.ch.Close()
}

func (r *responder) Continue(v interface{}) (transport.Channel, error) {
	return r.ch, r.respond(v, true)
}

func (r *responder) respond(v interface{}, continue_ bool) error {
	enc := r.c.Encoder(r.ch)
	r.header.Continue = continue_

	var e error
	var ok bool
	if e, ok = v.(error); ok {
		v = nil
	}
	if e != nil {
		var errStr = e.Error()
		r.header.Error = &errStr
	}

	err := enc.Encode(r.header)
	if err != nil {
		return err
	}

	err = enc.Encode(v)
	if err != nil {
		return err
	}

	return nil
}
