package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/progrium/qtalk-go/cmd/qtalk/cli"
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/fn"
	"github.com/progrium/qtalk-go/interop"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/rpc"
	cbor "github.com/progrium/qtalk-go/x/cbor/codec"
)

var checkCmd = &cli.Command{
	Usage: "check",
	Short: "check interop",
	Args:  cli.MinArgs(1),
	Run: func(ctx context.Context, args []string) {
		log.SetOutput(os.Stderr)

		var c codec.Codec = cbor.CBORCodec{}
		if os.Getenv("QTALK_CODEC") == "json" {
			log.Println("* Using JSON codec")
			c = codec.JSONCodec{}
		}

		// TODO: --self flag
		// TODO: quic endpoint

		path, err := exec.LookPath("sh")
		fatal(err)

		cmd := exec.Command(path, "-c", args[0])
		cmd.Stderr = os.Stderr
		wc, err := cmd.StdinPipe()
		if err != nil {
			fatal(err)
		}
		rc, err := cmd.StdoutPipe()
		if err != nil {
			fatal(err)
		}
		sess, err := mux.DialIO(wc, rc)
		if err != nil {
			fatal(err)
		}
		if err := cmd.Start(); err != nil {
			fatal(err)
		}

		srv := rpc.Server{
			Handler: fn.HandlerFrom(interop.CallbackService{}),
			Codec:   c,
		}
		go srv.Respond(sess, nil)

		caller := rpc.NewClient(sess, c)

		// Unary check
		// TODO: cycle different types of arg/ret
		var ret any
		_, err = caller.Call(ctx, "Unary", fn.Args{1, 2, 3}, &ret)
		fatal(err)
		fmt.Println("Unary:", ret)

		// Stream check
		// TODO: cycle different types of arg/ret
		resp, err := caller.Call(ctx, "Stream", nil, nil)
		fatal(err)
		go func() {
			fatal(resp.Send("Hello"))
			fatal(resp.Send(123))
			fatal(resp.Send(true))
			fatal(resp.Channel.CloseWrite())
		}()
		for {
			err = resp.Receive(&ret)
			if err != nil {
				break
			}
			fmt.Println("Stream:", ret)
		}

		// Bytes check
		// TODO: 1mb and 1gb
		data := make([]byte, 1024)
		rand.Read(data)
		resp, err = caller.Call(ctx, "Bytes", nil, nil)
		fatal(err)
		var buf bytes.Buffer
		go func() {
			io.Copy(resp.Channel, bytes.NewBuffer(data))
			resp.Channel.CloseWrite()
		}()
		io.Copy(&buf, resp.Channel)
		if buf.Len() != len(data) {
			log.Fatal("byte stream buffer does not match")
		}
		fmt.Println("Bytes:", buf.Len())

		// Error check
		// TODO: bad selector
		_, err = caller.Call(ctx, "Error", "test", nil)
		if err == nil {
			log.Fatal("expected error")
		}
		fmt.Println("Error:", strings.TrimPrefix(err.Error(), "remote: "))
	},
}
