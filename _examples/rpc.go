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
	sigkinds := []string{"md5", "sha1", "sha256"}
	ctx := context.TODO()
	jobs := make(chan Job)
	signatures := make(chan string)
	defer close(jobs)

	for _, kind := range sigkinds {
		local.Handle(kind, rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
			hasher(res, call, jobs, signatures)
		}))
	}

	remote.Handle(RunRPC, rpc.HandlerFunc(func(res rpc.Responder, call *rpc.Call) {
		p := &Ping{}

		if err := call.Receive(p); err != nil {
			res.Return(fmt.Errorf("ping err: %+v", err))
			return
		}

		pong, err := callCallbacks(ctx, call.Caller, reverse(p.Message), sigkinds...)
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
			_, err := cli.Call(ctx, RunRPC, ping, pong)
			if err != nil {
				fmt.Println("client call err: ", err)
				return err
			}

			fmt.Println(">> echo:     ", pong.Message)
			fmt.Println(" > md5:    ", pong.Args["md5"])
			fmt.Println(" > sha1:   ", pong.Args["sha1"])
			fmt.Println(" > sha256: ", pong.Args["sha256"])
			fmt.Print(">>> ")
		}
		return scanner.Err()
	}

	StartWorkers(3, jobs, signatures, func(job Job) (string, error) {
		hs := newHash(job.Selector)
		io.WriteString(hs, job.Message)
		return fmt.Sprintf("%x", hs.Sum(nil)), nil
	})

	fmt.Printf("[%s]\necho: hello.\n", RunRPC)
	err := stdinloop(local)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
	}
}

func hasher(res rpc.Responder, call *rpc.Call, jobs chan<- Job, results <-chan string) {
	p := &Ping{}
	if err := call.Receive(p); err != nil {
		res.Return(fmt.Errorf("ping err: %+v", err))
		return
	}

	jobs <- Job{Message: p.Message, Selector: call.Selector}

	if err := res.Return(&Ping{Message: <-results}); err != nil {
		// todo: find source of EOF
		fmt.Printf("hasher err: %+v\n", err)
	}
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

const RunRPC = "rpc"
