package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/rpc/rpctest"
)

type Ping struct {
	Message string `json:"msg"`
}

func main() {
	h := rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}
		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		res.Return(reverse(p.Message))
	})

	cli, _ := rpctest.NewPair(h, codec.JSONCodec{})

	fmt.Println("echo: hello.")
	err := stdinloop(cli)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
	}
}

func stdinloop(cli *rpc.Client) error {
	ctx := context.TODO()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print(">>> ")
	for scanner.Scan() {
		ping := &Ping{Message: scanner.Text()}
		pong := &Ping{}

		fmt.Println("send: ", ping.Message)
		res, err := cli.Call(ctx, "selector", ping, pong)
		if err != nil {
			return err
		}

		res.Receive(pong)
		fmt.Println("echo: ", pong.Message)
		fmt.Print(">>> ")
	}
	return scanner.Err()
}

func reverse(s string) *Ping {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return &Ping{Message: string(runes)}
}
