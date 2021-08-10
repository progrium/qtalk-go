package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/transport"
)

// $ go run _examples/*.go [example]
func main() {
	cli := newClient()
	defer cli.Close()

	for _, runner := range cliArgs() {
		runner(cli)
	}
}

// Ping is a sample datatype to pass through a json codec.
type Ping struct {
	Message string `json:"msg"`
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func cliArgs() map[string]func(*rpc.Client) {
	count := len(os.Args)
	if count < 2 {
		printRunnable()
		os.Exit(1)
	}

	h := make(map[string]func(*rpc.Client))
	for i := 1; i < count; i++ {
		if fn, ok := runnable[os.Args[i]]; ok {
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
	keys := make([]string, 0, len(runnable))
	for key := range runnable {
		keys = append(keys, key)
	}
	fmt.Printf("give an example to run:\n%s\n", strings.Join(keys, ", "))
}

func newClient() *rpc.Client {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := transport.DialIO(aw, ar)
	sessB, _ := transport.DialIO(bw, br)

	codec := codec.JSONCodec{}
	mux := rpc.NewRespondMux()
	mux.Handle(BiDirectionalRPC, newBiDirectionalRPCHandler())

	srv := &rpc.Server{
		Codec:   codec,
		Handler: mux,
	}
	go srv.Respond(sessA)

	return rpc.NewClient(sessB, codec)
}

var runnable map[string]func(*rpc.Client)

func init() {
	runnable = make(map[string]func(*rpc.Client))
	runnable[BiDirectionalRPC] = runBiDirectionalRPC
}
