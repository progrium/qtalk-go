package main

import (
	"context"
	"fmt"

	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
)

func runCallbacks(local, remote *peer.Peer) {
	ctx := context.TODO()

	remote.Handle(RunCallbacks, rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}
		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		res.Return(&Ping{Message: reverse(p.Message)})
	}))

	fmt.Printf("[%s]\necho: hello.\n", RunCallbacks)
	err := StdinLoop(func(ping, pong *Ping) error {
		if _, err := local.Call(ctx, RunCallbacks, ping, pong); err != nil {
			return err
		}

		fmt.Println("echo: ", pong.Message)
		return nil
	})

	if err != nil {
		fmt.Printf("err: %+v\n", err)
	}
}

const RunCallbacks = "callbacks"
