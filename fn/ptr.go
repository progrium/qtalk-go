package fn

import (
	"context"

	"github.com/progrium/qtalk-go/rpc"
	"github.com/rs/xid"
)

type Ptr struct {
	Ptr    string `json:"$fnptr" mapstructure:"$fnptr"`
	Caller rpc.Caller
	fn     interface{}
}

func (p *Ptr) Call(ctx context.Context, params, reply interface{}) (*rpc.Response, error) {
	return p.Caller.Call(ctx, p.Ptr, params, reply)
}

func Callback(fn interface{}) *Ptr {
	return &Ptr{
		Ptr: xid.New().String(),
		fn:  fn,
	}
}
