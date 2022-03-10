package rpc

import (
	"context"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
)

// A Caller is able to perform remote calls.
//
// Call makes synchronous calls to the remote selector passing args and putting the reply
// value in reply. Both args and reply can be nil. Args can be a channel of interface{}
// values for asynchronously streaming multiple values from another goroutine, however
// the call will still block until a response is sent. If there is an error making the call
// an error is returned, and if an error is returned by the remote handler a RemoteError
// is returned.
//
// A Response value is also returned for advanced operations. For example, you can check
// if the call is continued, meaning the underlying channel will be kept open for either
// streaming back more results or using the channel as a full duplex byte stream.
type Caller interface {
	Call(ctx context.Context, selector string, params, reply interface{}) (*Response, error)
}

// CallHeader is the first value encoded over the channel to make a call.
type CallHeader struct {
	Selector string
}

// Call is used on the responding side of a call and is passed to the handler.
// Call has a Caller so it can be used to make calls back to the calling side.
type Call struct {
	CallHeader

	Caller  Caller
	Decoder codec.Decoder
	Context context.Context
	ch      *mux.Channel
}

// Receive will decode an incoming value from the underlying channel. It can be
// called more than once when multiple values are expected, but should always be
// called once in a handler. It can be called with nil to discard the value.
func (c *Call) Receive(v interface{}) error {
	if v == nil {
		var discard []byte
		v = &discard
	}
	return c.Decoder.Decode(v)
}

// ResponseHeader is the value encoded over the channel to indicate a response.
type ResponseHeader struct {
	Error    *string
	Continue bool // after parsing response, keep stream open for whatever protocol
}

// Response is used on the calling side to represent a response and allow access
// to the ResponseHeader data, the reply value, the underlying channel, and methods
// to send or receive encoded values over the channel if Continue was set on the
// ResponseHeader.
type Response struct {
	ResponseHeader
	Reply   interface{}
	Channel *mux.Channel

	codec codec.Codec
}

// Send encodes a value over the underlying channel if it is still open.
func (r *Response) Send(v interface{}) error {
	return r.codec.Encoder(r.Channel).Encode(v)
}

// Receive decodes a value from the underlying channel if it is still open.
func (r *Response) Receive(v interface{}) error {
	return r.codec.Decoder(r.Channel).Decode(v)
}

// Responder is used by handlers to initiate a response and send values to the caller.
type Responder interface {
	// Return sends a return value, which can be an error, and closes the channel.
	Return(interface{}) error

	// Continue sets the response to keep the channel open after sending a return value,
	// and returns the underlying channel for you to take control of. If called, you
	// become responsible for closing the channel.
	Continue(interface{}) (*mux.Channel, error)

	// Send encodes a value over the underlying channel, but does not initiate a response,
	// so it must be used after calling Continue.
	Send(interface{}) error
}

type responder struct {
	responded bool
	header    *ResponseHeader
	ch        *mux.Channel
	c         codec.Codec
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
	r.responded = true
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
