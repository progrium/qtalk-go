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
	qquic "github.com/progrium/qtalk-go/x/quic"
	"github.com/quic-go/quic-go"
)

var vals = []any{
	100,
	true,
	"hello",
	map[string]any{"foo": "bar"},
	[]any{1, 2, 3},
}

var checkCmd = &cli.Command{
	Usage: "check",
	Short: "check interop",
	Run: func(ctx context.Context, args []string) {
		log.SetOutput(os.Stderr)

		var c codec.Codec = cbor.CBORCodec{}
		if os.Getenv("QTALK_CODEC") == "json" {
			log.Println("* Using JSON codec")
			c = codec.JSONCodec{}
		}

		var cmd *exec.Cmd
		var sess mux.Session

		if len(args) == 0 {
			// self check
			path, err := os.Executable()
			fatal(err)
			cmd = exec.Command(path, "interop")
		} else if !strings.HasPrefix(args[0], "udp://") {
			// check against subprocess
			path, err := exec.LookPath("sh")
			fatal(err)
			cmd = exec.Command(path, "-c", args[0])
		}

		if cmd != nil {
			cmd.Stderr = os.Stderr
			wc, err := cmd.StdinPipe()
			if err != nil {
				fatal(err)
			}
			rc, err := cmd.StdoutPipe()
			if err != nil {
				fatal(err)
			}
			sess, err = mux.DialIO(wc, rc)
			if err != nil {
				fatal(err)
			}
			if err := cmd.Start(); err != nil {
				fatal(err)
			}
			defer func() {
				cmd.Process.Signal(os.Interrupt)
				cmd.Wait()
			}()
		} else {
			// check against remote quic endpoint
			cfg := defaultTLSConfig.Clone()
			cfg.InsecureSkipVerify = true
			conn, err := quic.DialAddr(strings.TrimPrefix(args[0], "udp://"), cfg, nil)
			fatal(err)
			sess = qquic.New(conn)
		}

		defer sess.Close()

		srv := rpc.Server{
			Handler: fn.HandlerFrom(interop.CallbackService{}),
			Codec:   c,
		}
		go srv.Respond(sess, nil)

		caller := rpc.NewClient(sess, c)
		var ret any
		var err error

		// Error check
		_, err = caller.Call(ctx, "Error", "test", nil)
		if err == nil {
			log.Fatal("expected error")
		}
		fmt.Println("Error:", strings.TrimPrefix(err.Error(), "remote: "))
		_, err = caller.Call(ctx, "BadSelector", "test", nil)
		if err == nil {
			log.Fatal("expected error")
		}
		fmt.Println("Error:", strings.TrimPrefix(err.Error(), "remote: "))

		// Unary check
		for _, v := range vals {
			_, err = caller.Call(ctx, "Unary", v, &ret)
			fatal(err)
			fmt.Println("Unary:", v, ret)
		}

		// Stream check
		resp, err := caller.Call(ctx, "Stream", nil, nil)
		fatal(err)
		go func() {
			for _, v := range vals {
				fatal(resp.Send(v))
			}
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
		// 1 byte, 1kb, 1mb
		for _, v := range []int{1, 1024, 1 << 20} {
			data := make([]byte, v)
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
		}
	},
}
