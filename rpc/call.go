package rpc

import (
	"context"

	"github.com/progrium/qtalk-go/codec"
)

type Call struct {
	Selector string

	Caller  Caller
	Decoder codec.Decoder
	Context context.Context
}

func (c *Call) Receive(v interface{}) error {
	return c.Decoder.Decode(v)
}
