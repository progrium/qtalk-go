package main

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"

	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
)

func runRPC(local, remote *peer.Peer) error {
	// this local peer exposes selectors for hashing.
	selectors := []string{"md5", "sha1", "sha256"}
	ctx := context.TODO()
	jobs := make(chan Job)
	signatures := make(chan string)
	defer close(jobs)

	// teach local peer how to handle hash selectors
	for _, selector := range selectors {
		local.Handle(selector, rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
			p := &Ping{}
			if err := call.Receive(p); err != nil {
				res.Return(fmt.Errorf("local recv err: %+v", err))
				return
			}

			jobs <- Job{Message: p.Message, Selector: call.Selector}

			if err := res.Return(&Ping{Message: <-signatures}); err != nil {
				// todo: find source of EOF
				fmt.Printf("hasher err: %+v\n", err)
			}
		}))
	}

	// teach remote peer to handle rpc selector
	remote.Handle("rpc", rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}

		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("remote recv err: %+v", err))
			return
		}

		// reverse the given message, pass to all selectors for hashing
		pong, err := CallCallbacks(ctx, call.Caller, reverse(p.Message), selectors...)
		if err != nil {
			res.Return(fmt.Errorf("remote callback err: %+v", err))
		}

		if err := res.Return(pong); err != nil {
			fmt.Printf("remote return err: %+v\n", err)
		}
	}))

	StartWorkers(3, jobs, signatures, func(job Job) (string, error) {
		hs := newHash(job.Selector)
		io.WriteString(hs, job.Message)
		return fmt.Sprintf("%x", hs.Sum(nil)), nil
	})

	fmt.Println("[rpc example]\necho: hello.")
	return StdinLoop(func(ping, pong *Ping) error {
		if _, err := local.Call(ctx, "rpc", ping, pong); err != nil {
			fmt.Println("local call err: ", err)
			return err
		}

		fmt.Println("<< echo:   ", pong.Message)
		fmt.Println(" < md5:    ", pong.Args["md5"])
		fmt.Println(" < sha1:   ", pong.Args["sha1"])
		fmt.Println(" < sha256: ", pong.Args["sha256"])
		return nil
	})
}

func newHash(selector string) hash.Hash {
	switch selector {
	case "md5":
		return md5.New()
	case "sha1":
		return sha1.New()
	case "sha256":
		return sha256.New()
	}
	return nil
}
