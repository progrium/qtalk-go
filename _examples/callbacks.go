package main

import (
	"context"
	"fmt"

	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
)

func runCallbacks(local, remote *peer.Peer) error {
	ctx := context.TODO()

	remote.Handle("callbacks", rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}
		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		res.Return(&Ping{Message: reverse(p.Message)})
	}))

	fmt.Println("[callbacks example]\necho: hello.")
	return StdinLoop(func(ping, pong *Ping) error {
		if _, err := local.Call(ctx, "callbacks", ping, pong); err != nil {
			return err
		}

		fmt.Println("echo: ", pong.Message)
		return nil
	})
}
