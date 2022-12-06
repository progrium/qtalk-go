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
func (c *Client) Call(ctx context.Context, selector string, args any, replies ...any) (*Response, error) {
	ch, err := c.Session.Open(ctx)
	if err != nil {
		return nil, err
	}
	// If the context is cancelled before the call completes, call Close() to
	// abort the current operation.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			ch.Close()
		case <-done:
		}
	}()
	resp, err := call(ctx, ch, c.codec, selector, args, replies...)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return resp, ctxErr
	}
	return resp, err
}

func call(ctx context.Context, ch mux.Channel, cd codec.Codec, selector string, args any, replies ...any) (*Response, error) {
	framer := &FrameCodec{Codec: cd}
	enc := framer.Encoder(ch)
	dec := framer.Decoder(ch)

	// request
	err := enc.Encode(CallHeader{
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
		codec:          framer,
	}
	if len(replies) == 1 {
		resp.Reply = replies[0]
	} else if len(replies) > 1 {
		resp.Reply = replies
	}
	if resp.Error != nil {
		return resp, RemoteError(*resp.Error)
	}

	if resp.Reply == nil {
		// read into throwaway buffer
		var buf []byte
		dec.Decode(&buf)
	} else {
		for _, r := range replies {
			if err := dec.Decode(r); err != nil {
				return resp, err
			}
		}
	}

	return resp, nil
}
