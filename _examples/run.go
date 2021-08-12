package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport"
)

// $ go run _examples/*.go [example]
func main() {
	peer1, peer2 := newPeers()
	defer peer1.Close()
	defer peer2.Close()
	go peer1.Respond()
	go peer2.Respond()

	for example, run := range cliArgs() {
		if err := run(peer1, peer2); err != nil {
			log.Fatalf("error running %s example: %+v", example, err)
		}
	}
}

// Ping represents a simple value.
type Ping struct {
	Message string            `json:"msg"`
	Args    map[string]string `json:"args"`
}

// CallCallbacks passes the message to the caller with the given selectors.
func CallCallbacks(ctx context.Context, caller rpc.Caller, msg string, selectors ...string) (*Ping, error) {
	pong := &Ping{Message: msg, Args: make(map[string]string)}
	for _, selector := range selectors {
		selpong := &Ping{}
		_, err := caller.Call(ctx, selector, pong, selpong)
		if err != nil {
			return pong, err
		}

		pong.Args[selector] = selpong.Message
	}
	return pong, nil
}

// Job represents a task for a peer's background workers.
type Job struct {
	Message  string
	Selector string
}

// StartWorkers runs a number of workers with the given fn in a goroutine.
func StartWorkers(num int, jobs <-chan Job, results chan<- string, fn func(job Job) (string, error)) {
	for id := 1; id <= num; id++ {
		go RunWorker(id, jobs, results, fn)
	}
}

// RunWorker runs a task from the job channel with the given fn.
func RunWorker(id int, jobs <-chan Job, results chan<- string, fn func(job Job) (string, error)) {
	for job := range jobs {
		start := time.Now()
		fmt.Printf("worker %d: sel %q, job %q ", id, job.Selector, job.Message)
		res, err := fn(job)
		if err != nil {
			fmt.Printf("err %+v ", err)
		}
		fmt.Printf("// %s\n", time.Since(start))
		results <- res
	}
}

// StdinLoop passes new console messages to the given fn.
func StdinLoop(fn func(ping, pong *Ping) error) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(">>> ")

	var ping, pong *Ping
	var err error
	for scanner.Scan() {
		ping = &Ping{Message: scanner.Text()}
		pong = &Ping{}

		fmt.Println("send: ", ping.Message)
		err = fn(ping, pong)
		fmt.Print(">>> ")

		if err != nil {
			return err
		}
	}
	return scanner.Err()
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func cliArgs() map[string]func(*peer.Peer, *peer.Peer) error {
	count := len(os.Args)
	if count < 2 {
		printRunnable()
		os.Exit(1)
	}

	h := make(map[string]func(*peer.Peer, *peer.Peer) error)
	for i := 1; i < count; i++ {
		if fn, ok := examples[os.Args[i]]; ok {
			h[os.Args[i]] = fn
			continue
		}
		fmt.Fprintf(os.Stderr, "not runnable: %s\n", os.Args[i])
	}

	if len(h) < 1 {
		printRunnable()
		os.Exit(1)
	}
	return h
}

func printRunnable() {
	keys := make([]string, 0, len(examples))
	for key := range examples {
		keys = append(keys, key)
	}
	fmt.Printf("give an example to run:\n%s\n", strings.Join(keys, ", "))
}

func newPeers() (*peer.Peer, *peer.Peer) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := transport.DialIO(aw, ar)
	sessB, _ := transport.DialIO(bw, br)

	js := codec.JSONCodec{}
	return peer.New(sessA, js), peer.New(sessB, js)
}

var examples map[string]func(*peer.Peer, *peer.Peer) error

func init() {
	examples = make(map[string]func(*peer.Peer, *peer.Peer) error)
	examples["callbacks"] = runCallbacks
	examples["rpc"] = runRPC
	examples["streaming"] = runStreaming
}
