package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
)

func runCallbacks(local, remote *peer.Peer) {
	remote.Handle(RunCallbacks, rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}
		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		res.Return(&Ping{Message: reverse(p.Message)})
	}))

	stdinloop := func(cli *peer.Peer) error {
		scanner := bufio.NewScanner(os.Stdin)

		fmt.Print(">>> ")
		for scanner.Scan() {
			ping := &Ping{Message: scanner.Text()}
			pong := &Ping{}

			fmt.Println("send: ", ping.Message)
			res, err := cli.Call(context.TODO(), RunCallbacks, ping, pong)
			if err != nil {
				return err
			}

			res.Receive(pong)
			fmt.Println("echo: ", pong.Message)
			fmt.Print(">>> ")
		}
		return scanner.Err()
	}

	fmt.Printf("[%s]\necho: hello.\n", RunCallbacks)
	err := stdinloop(local)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
	}
}

const RunCallbacks = "callbacks"
