package rpc

import (
	"context"
	"io"
	"io/ioutil"
	"testing"
)

func TestProxyHandlerUnaryRPC(t *testing.T) {
	ctx := context.Background()

	backmux := NewRespondMux()
	backmux.Handle("simple", HandlerFunc(func(r Responder, c *Call) {
		r.Return("simple")
	}))

	backend, _ := newTestPair(backmux)
	defer backend.Close()

	frontmux := NewRespondMux()
	frontmux.Handle("", ProxyHandler(backend))

	client, _ := newTestPair(frontmux)
	defer client.Close()

	var out interface{}
	_, err := client.Call(ctx, "simple", nil, &out)
	fatal(t, err)
	if out != "simple" {
		t.Fatal("unexpected return:", out)
	}
}

func TestProxyHandlerBytestream(t *testing.T) {
	ctx := context.Background()

	backmux := NewRespondMux()
	backmux.Handle("echo", HandlerFunc(func(r Responder, c *Call) {
		c.Receive(nil)
		ch, err := r.Continue(nil)
		fatal(t, err)
		io.Copy(ch, ch)
		ch.Close()
	}))

	backend, _ := newTestPair(backmux)
	defer backend.Close()

	frontmux := NewRespondMux()
	frontmux.Handle("", ProxyHandler(backend))

	client, _ := newTestPair(frontmux)
	defer client.Close()

	resp, err := client.Call(ctx, "echo", nil, nil)
	fatal(t, err)
	_, err = io.WriteString(resp.Channel, "Hello world")
	fatal(t, err)
	fatal(t, resp.Channel.CloseWrite())
	b, err := ioutil.ReadAll(resp.Channel)
	fatal(t, err)
	if string(b) != "Hello world" {
		t.Fatal("unexpected return data:", string(b))
	}
}
