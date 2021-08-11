package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/peer"
	"github.com/progrium/qtalk-go/transport"
)

// $ go run _examples/*.go [example]
func main() {
	peer1, peer2 := newPeers()
	defer peer1.Close()
	defer peer2.Close()
	go peer1.Respond()
	go peer2.Respond()

	for _, runner := range cliArgs() {
		runner(peer1, peer2)
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

func cliArgs() map[string]func(*peer.Peer, *peer.Peer) {
	count := len(os.Args)
	if count < 2 {
		printRunnable()
		os.Exit(1)
	}

	h := make(map[string]func(*peer.Peer, *peer.Peer))
	for i := 1; i < count; i++ {
		if fn, ok := runnable2[os.Args[i]]; ok {
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
	keys := make([]string, 0, len(runnable2))
	for key := range runnable2 {
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

var runnable2 map[string]func(*peer.Peer, *peer.Peer)

func init() {
	runnable2 = make(map[string]func(*peer.Peer, *peer.Peer))
	runnable2[RunCallbacks] = runCallbacks
	runnable2[RunRPC] = runRPC
}
