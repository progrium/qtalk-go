package interop

import (
	"io"
	"log"

	"github.com/progrium/qtalk-go/rpc"
)

type CallbackService struct{}

func (s CallbackService) Unary(resp rpc.Responder, call *rpc.Call) {
	var params any
	if err := call.Receive(&params); err != nil {
		log.Println(err)
		return
	}
	if err := resp.Return(params); err != nil {
		log.Println(err)
	}
}

func (s CallbackService) Stream(resp rpc.Responder, call *rpc.Call) {
	var v any
	if err := call.Receive(&v); err != nil {
		log.Println(err)
		return
	}
	ch, err := resp.Continue(v)
	if err != nil {
		log.Println(err)
		return
	}
	defer ch.Close()
	for err == nil {
		err = call.Receive(&v)
		if err == nil {
			err = resp.Send(v)
		}
	}
}

func (s CallbackService) Bytes(resp rpc.Responder, call *rpc.Call) {
	var params any
	if err := call.Receive(&params); err != nil {
		log.Println(err)
		return
	}
	ch, err := resp.Continue(params)
	if err != nil {
		log.Println(err)
		return
	}
	defer ch.Close()
	io.Copy(ch, call)
}
