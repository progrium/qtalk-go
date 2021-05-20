package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/transport"
)

// RemoteError represents an error that has been returned from
// the remote side of the RPC connection.
type RemoteError string

func (e RemoteError) Error() string {
	return fmt.Sprintf("remote: %s", string(e))
}

type Client struct {
	session transport.Session
	codec   codec.Codec
}

func NewClient(session transport.Session, codec codec.Codec) *Client {
	return &Client{
		session: session,
		codec:   codec,
	}
}

func (c *Client) Call(ctx context.Context, selector string, args, reply interface{}) (*Response, error) {
	ch, err := c.session.Open(ctx)
	if err != nil {
		return nil, err
	}

	framer := &codec.FrameCodec{Codec: c.codec}
	enc := framer.Encoder(ch)
	dec := framer.Decoder(ch)

	// request
	err = enc.Encode(Call{
		Selector: selector,
	})
	if err != nil {
		ch.Close()
		return nil, err
	}

	err = enc.Encode(args)
	if err != nil {
		ch.Close()
		return nil, err
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
		if err := dec.Decode(&buf); err != nil {
			if errors.Is(err, io.EOF) {
				return resp, nil
			}
			return resp, err
		}
	} else {
		// TODO: timeout
		if err := dec.Decode(resp.Reply); err != nil {
			return resp, err
		}
	}

	return resp, nil
}
