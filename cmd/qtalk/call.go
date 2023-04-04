package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/progrium/clon-go"
	"github.com/progrium/qtalk-go/cmd/qtalk/cli"
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/talk"
)

var callCmd = &cli.Command{
	Usage: "call",
	Short: "call a remote function",
	Args:  cli.MinArgs(1),
	Run: func(ctx context.Context, args []string) {
		log.SetOutput(os.Stderr)
		u, err := url.Parse(args[0])
		if err != nil {
			log.Fatal(err)
		}

		var sargs any
		if len(args) > 1 {
			sargs, err = clon.Parse(args[1:])
			if err != nil {
				log.Fatal(err)
			}
		}

		peer, err := talk.Dial(u.Scheme, u.Host, codec.JSONCodec{})
		if err != nil {
			log.Fatal(err)
		}
		defer peer.Close()

		var ret any
		_, err = peer.Call(context.Background(), u.Path, sargs, &ret)
		if err != nil {
			log.Fatal(err)
		}

		b, err := json.MarshalIndent(ret, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
	},
}
