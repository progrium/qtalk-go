package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport"
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

	cli, _ := newpair(h, codec.JSONCodec{})
	defer cli.Close()

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
		res, err := cli.Call(ctx, "", ping, pong)
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

func newpair(handler rpc.Handler, codec codec.Codec) (*rpc.Client, *rpc.Server) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := transport.DialIO(aw, ar)
	sessB, _ := transport.DialIO(bw, br)

	srv := &rpc.Server{
		Codec:   codec,
		Handler: handler,
	}
	go srv.Respond(sessA)

	return rpc.NewClient(sessB, codec), srv
}
