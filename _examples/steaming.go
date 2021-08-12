package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
)

func runStreaming(local, remote *peer.Peer) error {
	ctx := context.TODO()

	remote.Handle("streaming", rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		hs := md5.New()
		var msg string

		for {
			if err := call.Receive(&msg); err != nil {
				if err == io.EOF {
					break
				}
				res.Return(fmt.Errorf("remote recv err: %+v", err))
				return
			}
			fmt.Println("remote recv", msg)
			io.WriteString(hs, msg)
		}

		ch, err := res.Continue(nil)
		if err != nil {
			res.Return(fmt.Errorf("remote continue err: %+v", err))
			return
		}
		defer ch.Close()

		for _, char := range hex.EncodeToString(hs.Sum(nil)) {
			if _, err := ch.Write([]byte(string(char))); err != nil {
				res.Return(fmt.Errorf("remote write err: %+v", err))
				return
			}
		}
	}))

	fmt.Println("[streaming example]\necho: hello.")
	return StdinLoop(func(ping, pong *Ping) error {
		sender := make(chan interface{})
		go func() {
			for _, char := range ping.Message {
				sender <- string(char)
			}
			close(sender)
		}()

		res, err := local.Call(ctx, "streaming", sender, nil)
		if err != nil {
			return err
		}

		fmt.Println("<< echo: ")

		var recv string
		for {
			if err := res.Receive(&recv); err != nil {
				return err
			}
			fmt.Println(" < ", recv)
		}

		return nil
	})
}
