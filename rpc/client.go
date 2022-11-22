package rpc

import (
	"context"
	"fmt"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
)

// RemoteError is an error that has been returned from
// the remote side of the RPC connection.
type RemoteError string

func (e RemoteError) Error() string {
	return fmt.Sprintf("remote: %s", string(e))
}

// Client wraps a session and codec to make RPC calls over the session.
type Client struct {
	mux.Session
	codec codec.Codec
}

// NewClient takes a session and codec to make a client for making RPC calls.
func NewClient(session mux.Session, codec codec.Codec) *Client {
	return &Client{
		Session: session,
		codec:   codec,
	}
}

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
func (c *Client) Call(ctx context.Context, selector string, args, reply interface{}) (*Response, error) {
	ch, err := c.Session.Open(ctx)
	if err != nil {
		return nil, err
	}

	framer := &FrameCodec{Codec: c.codec}
	enc := framer.Encoder(ch)
	dec := framer.Decoder(ch)

	// request
	err = enc.Encode(CallHeader{
		Selector: selector,
	})
	if err != nil {
		ch.Close()
		return nil, err
	}

	argCh, isChan := args.(chan interface{})
	switch {
	case isChan:
		for arg := range argCh {
			if err := enc.Encode(arg); err != nil {
				ch.Close()
				return nil, err
			}
		}
	default:
		if err := enc.Encode(args); err != nil {
			ch.Close()
			return nil, err
		}
	}

	// response
	// TODO: timeout
	var header ResponseHeader
	err = dec.Decode(&header)
	if err != nil {
		ch.Close()
		return nil, err
	}

	if !header.Continue {
		defer ch.Close()
	}

	resp := &Response{
		ResponseHeader: header,
		Channel:        ch,
		Reply:          reply,
		codec:          framer,
	}
	if resp.Error != nil {
		return resp, RemoteError(*resp.Error)
	}

	if reply == nil {
		// read into throwaway buffer
		var buf []byte
		dec.Decode(&buf)
	} else {
		// TODO: timeout
		if err := dec.Decode(resp.Reply); err != nil {
			return resp, err
		}
	}

	return resp, nil
}
