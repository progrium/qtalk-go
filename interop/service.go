package interop

import (
	"context"
	"errors"
	"io"
	"log"

	"github.com/progrium/qtalk-go/rpc"
)

type InteropService struct{}

func (s InteropService) Unary(resp rpc.Responder, call *rpc.Call) {
	var params any
	if err := call.Receive(&params); err != nil {
		log.Println(err)
		return
	}
	ctx := context.Background()
	var ret any
	_, err := call.Caller.Call(ctx, "Unary", params, &ret)
	if err != nil {
		log.Println(err)
		return
	}
	if err := resp.Return(ret); err != nil {
		log.Println(err)
	}
}

func (s InteropService) Stream(resp rpc.Responder, call *rpc.Call) {
	var params any
	if err := call.Receive(&params); err != nil {
		log.Println(err)
		return
	}
	ctx := context.Background()
	var ret any
	stream, err := call.Caller.Call(ctx, "Stream", params, &ret)
	if err != nil {
		log.Println(err)
		return
	}
	ch, err := resp.Continue(ret)
	if err != nil {
		log.Println(err)
		return
	}
	defer ch.Close()
	go func() {
		var v any
		var err error
		for {
			err = call.Receive(&v)
			if err != nil {
				break
			}
			err = stream.Send(v)
			if err != nil {
				break
			}
		}
		stream.Channel.CloseWrite()
	}()
	var v any
	for {
		err = stream.Receive(&v)
		if err != nil {
			break
		}
		err = resp.Send(v)
		if err != nil {
			break
		}
	}
}

func (s InteropService) Bytes(resp rpc.Responder, call *rpc.Call) {
	var params any
	if err := call.Receive(&params); err != nil {
		log.Println(err)
		return
	}
	ctx := context.Background()
	var ret any
	stream, err := call.Caller.Call(ctx, "Bytes", params, &ret)
	if err != nil {
		log.Println(err)
		return
	}
	ch, err := resp.Continue(ret)
	if err != nil {
		log.Println(err)
		return
	}
	defer ch.Close()
	go func() {
		io.Copy(stream.Channel, call)
		stream.Channel.CloseWrite()
	}()
	io.Copy(ch, stream.Channel)
}

func (s InteropService) Error(resp rpc.Responder, call *rpc.Call) {
	var text string
	if err := call.Receive(&text); err != nil {
		log.Println(err)
		return
	}
	if err := resp.Return(errors.New(text)); err != nil {
		log.Println(err)
	}
}
