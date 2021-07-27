package rpc

import (
	"context"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
)

type Caller interface {
	Call(ctx context.Context, selector string, params, reply interface{}) (*Response, error)
}

type Call struct {
	Selector string

	Caller  Caller
	Decoder codec.Decoder
}

func (c *Call) Receive(v interface{}) error {
	return c.Decoder.Decode(v)
}

type ResponseHeader struct {
	Error    *string
	Continue bool // after parsing response, keep stream open for whatever protocol
}

type Response struct {
	ResponseHeader
	Reply   interface{}
	Channel *mux.Channel

	codec codec.Codec
}

func (r *Response) Send(v interface{}) error {
	return r.codec.Encoder(r.Channel).Encode(v)
}

func (r *Response) Receive(v interface{}) error {
	return r.codec.Decoder(r.Channel).Decode(v)
}

type Responder interface {
	Header() *ResponseHeader
	Return(interface{}) error
	Continue(interface{}) (*mux.Channel, error)
	Send(interface{}) error
}

type responder struct {
	header *ResponseHeader
	ch     *mux.Channel
	c      codec.Codec
}

func (r *responder) Header() *ResponseHeader {
	return r.header
}

func (r *responder) Send(v interface{}) error {
	return r.c.Encoder(r.ch).Encode(v)
}

func (r *responder) Return(v interface{}) error {
	return r.respond(v, false)
}

func (r *responder) Continue(v interface{}) (*mux.Channel, error) {
	return r.ch, r.respond(v, true)
}

func (r *responder) respond(v interface{}, continue_ bool) error {
	r.header.Continue = continue_

	// if v is error, set v to nil
	// and put error in header
	var e error
	var ok bool
	if e, ok = v.(error); ok {
		v = nil
	}
	if e != nil {
		var errStr = e.Error()
		r.header.Error = &errStr
	}

	if err := r.Send(r.header); err != nil {
		return err
	}

	if err := r.Send(v); err != nil {
		return err
	}

	if !continue_ {
		return r.ch.Close()
	}

	return nil
}
