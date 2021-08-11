package main

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
)

func runRPC(local, remote *peer.Peer) {
	ctx := context.TODO()

	local.Handle("md5", rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		hasher(res, call, md5.New)
	}))
	local.Handle("sha1", rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		hasher(res, call, sha1.New)
	}))
	local.Handle("sha256", rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		hasher(res, call, sha256.New)
	}))

	remote.Handle(RunRPC, rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}

		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		pong, err := callCallbacks(ctx, call.Caller, reverse(p.Message), "md5", "sha1", "sha256")
		if err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
		}

		if err := res.Return(pong); err != nil {
			res.Return(fmt.Errorf("error returning: %+v", err))
		}
	}))

	stdinloop := func(cli *peer.Peer) error {
		scanner := bufio.NewScanner(os.Stdin)

		fmt.Print(">>> ")
		for scanner.Scan() {
			ping := &Ping{Message: scanner.Text()}
			pong := &Ping{}

			fmt.Println("send: ", ping.Message)
			res, err := cli.Call(ctx, RunRPC, ping, pong)
			if err != nil {
				fmt.Println("client call err: ", err)
				return err
			}

			// todo: find source of EOF
			if err := res.Receive(pong); err != nil && err != io.EOF {
				fmt.Println("client recv err: ", err)
				return err
			}

			fmt.Println("echo    : ", pong.Message)
			fmt.Println("  md5   : ", pong.Args["md5"])
			fmt.Println("  sha1  : ", pong.Args["sha1"])
			fmt.Println("  sha256: ", pong.Args["sha256"])
			fmt.Print(">>> ")
		}
		return scanner.Err()
	}

	fmt.Printf("[%s]\necho: hello.\n", RunRPC)
	err := stdinloop(local)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
	}
}

func hasher(res rpc.Responder, call *rpc.Call, new func() hash.Hash) {
	p := &Ping{}
	if err := call.Receive(p); err != nil {
		res.Return(fmt.Errorf("ping err: %+v", err))
		return
	}
	hs := new()
	io.WriteString(hs, p.Message)

	if err := res.Return(&Ping{Message: fmt.Sprintf("%x", hs.Sum(nil))}); err != nil {
		fmt.Printf("hasher err: %+v\n", err)
	}
}

const RunRPC = "rpc"
