package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/progrium/qtalk-go/mux"
	qtalkwebrpc "github.com/progrium/qtalk-go/x/webrtc"
)

// Usage:
// Start one terminal with an offer, and another with "-a" to answer:
// $ go run main.go
// $ go run main.go -a
//
// Copy the SDP from the offer output to the answer.
// Copy the answer SDP to the offer.

var isAnswer = flag.Bool("a", false, "answer mode")

func fatal(err error, args ...any) {
	if err != nil {
		log.Fatal(append(args, err)...)
	}
}

func main() {
	flag.Parse()
	cfg := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	var peer *webrtc.PeerConnection
	var err error
	if *isAnswer {
		log.Println("paste SDP from offer:")
		peer, err = qtalkwebrpc.Answer(cfg, getSDP())
	} else {
		log.Println("creating offer...")
		peer, err = qtalkwebrpc.Offer(cfg)
	}
	fatal(err)

	// For simple case, wait to gather ICE candidates before printing the SDP
	log.Println("gathering ICE candidates...")

	// peer.OnICECandidate(func(c *webrtc.ICECandidate) {
	// 	go func() {
	// 		log.Println("got ICE candidate, latest SDP:")
	// 		printDesc(peer)
	// 	}()
	// })

	<-webrtc.GatheringCompletePromise(peer)
	log.Println("gathering complete, SDP is:")
	printDesc(peer)

	if !*isAnswer {
		log.Println("paste SDP from answer:")
		fatal(peer.SetRemoteDescription(getSDP()))
	}

	sess := qtalkwebrpc.New(peer)
	defer sess.Close()
	log.Println("got session")
	var ch mux.Channel
	if *isAnswer {
		ch, err = sess.Open(context.Background())
	} else {
		ch, err = sess.Accept()
	}
	fatal(err)
	defer ch.Close()
	log.Println("got channel")

	tick := time.Tick(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		r := bufio.NewReader(ch)
		for {
			line, err := r.ReadString('\n')
			fatal(err, "read line")
			log.Println("read: ", line)
		}
	}()

	for {
		select {
		case <-tick:
			log.Println("sending: ping")
			_, err := ch.Write([]byte("ping\n"))
			fatal(err, "send ping")
		case <-ctx.Done():
			return
		}
	}
}

func printDesc(peer *webrtc.PeerConnection) {
	desc := peer.LocalDescription()
	b, err := json.Marshal(desc)
	fatal(err)
	s := base64.URLEncoding.EncodeToString(b)
	for len(s) > 80 {
		fmt.Println(s[:80])
		s = s[80:]
	}
	fmt.Println(s)
	fmt.Println()
}

func getSDP() webrtc.SessionDescription {
	r := bufio.NewReader(os.Stdin)
	var lines []string
	for {
		b, err := r.ReadString('\n')
		fatal(err)
		if b == "\n" {
			break
		}
		lines = append(lines, b)
	}
	s, err := base64.URLEncoding.DecodeString(strings.Join(lines, ""))
	fatal(err)
	log.Println("json:", string(s))
	var desc webrtc.SessionDescription
	fatal(json.Unmarshal(s, &desc))
	return desc
}
