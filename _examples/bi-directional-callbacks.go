package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/progrium/qtalk-go/rpc"
)

func newBiDirectionalCallbacksHandler() rpc.Handler {
	return rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}
		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		res.Return(&Ping{Message: reverse(p.Message)})
	})
}

func runBiDirectionalCallbacks(cli *rpc.Client) {
	stdinloop := func(cli *rpc.Client) error {
		scanner := bufio.NewScanner(os.Stdin)

		fmt.Print(">>> ")
		for scanner.Scan() {
			ping := &Ping{Message: scanner.Text()}
			pong := &Ping{}

			fmt.Println("send: ", ping.Message)
			res, err := cli.Call(context.TODO(), BiDirectionalRPC, ping, pong)
			if err != nil {
				return err
			}

			res.Receive(pong)
			fmt.Println("echo: ", pong.Message)
			fmt.Print(">>> ")
		}
		return scanner.Err()
	}

	fmt.Printf("[%s]\necho: hello.\n", BiDirectionalRPC)
	err := stdinloop(cli)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
	}
}

const BiDirectionalCallbacks = "bi-directional-callbacks"
