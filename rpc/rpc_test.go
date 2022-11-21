package rpc

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
)

func fatal(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newTestPair(handler Handler) (*Client, *Server) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	sessA, _ := mux.DialIO(aw, ar)
	sessB, _ := mux.DialIO(bw, br)

	srv := &Server{
		Codec:   codec.JSONCodec{},
		Handler: handler,
	}
	go srv.Respond(sessA, nil)

	return NewClient(sessB, codec.JSONCodec{}), srv
}

func TestServerNoCodec(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("did not panic from unset codec")
		}
	}()

	ar, _ := io.Pipe()
	_, aw := io.Pipe()
	sessA, _ := mux.DialIO(aw, ar)

	srv := &Server{
		Handler: NotFoundHandler(),
	}
	go sessA.Close()
	srv.Respond(sessA, nil)
}

func TestRespondMux(t *testing.T) {
	ctx := context.Background()

	t.Run("selector mux", func(t *testing.T) {
		mux := NewRespondMux()
		mux.Handle("foo", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foo")
		}))
		mux.Handle("bar", HandlerFunc(func(r Responder, c *Call) {
			r.Return("bar")
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "foo", nil, &out)
		fatal(t, err)
		if out != "foo" {
			t.Fatal("unexpected return:", out)
		}

		_, err = client.Call(ctx, "bar", nil, &out)
		fatal(t, err)
		if out != "bar" {
			t.Fatal("unexpected return:", out)
		}
	})

	t.Run("selector not found error", func(t *testing.T) {
		mux := NewRespondMux()
		mux.Handle("foo", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foo")
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "baz", nil, &out)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != nil {
			rErr, ok := err.(RemoteError)
			if !ok {
				t.Fatal("unexpected error:", err)
			}
			if rErr.Error() != "remote: not found: /baz" {
				t.Fatal("unexpected error:", rErr)
			}
		}
	})

	t.Run("default handler mux", func(t *testing.T) {
		mux := NewRespondMux()
		mux.Handle("foo", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foo")
		}))
		mux.Handle("", HandlerFunc(func(r Responder, c *Call) {
			r.Return(fmt.Errorf("default"))
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "baz", nil, &out)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != nil {
			rErr, ok := err.(RemoteError)
			if !ok {
				t.Fatal("unexpected error:", err)
			}
			if rErr.Error() != "remote: default" {
				t.Fatal("unexpected error:", rErr)
			}
		}

		_, err = client.Call(ctx, "foo", nil, &out)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if out != "foo" {
			t.Fatal("unexpected return:", out)
		}
	})

	t.Run("sub muxing", func(t *testing.T) {
		mux := NewRespondMux()
		submux := NewRespondMux()
		mux.Handle("foo.bar", submux)
		mux.Handle("", HandlerFunc(func(r Responder, c *Call) {
			r.Return(fmt.Errorf("default"))
		}))
		submux.Handle("baz", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foobarbaz")
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "foo.bar.baz", nil, &out)
		fatal(t, err)
		if out != "foobarbaz" {
			t.Fatal("unexpected return:", out)
		}
	})

	t.Run("selector normalizing", func(t *testing.T) {
		mux := NewRespondMux()
		mux.Handle("foo.bar", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foobar")
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "/foo/bar", nil, &out)
		fatal(t, err)
		if out != "foobar" {
			t.Fatal("unexpected return:", out)
		}
	})

	t.Run("selector catchall", func(t *testing.T) {
		mux := NewRespondMux()
		mux.Handle("foo.bar.", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foobar")
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "foo.bar.baz", nil, &out)
		fatal(t, err)
		if out != "foobar" {
			t.Fatal("unexpected return:", out)
		}
	})

	t.Run("remove handler", func(t *testing.T) {
		mux := NewRespondMux()
		mux.Handle("foo", HandlerFunc(func(r Responder, c *Call) {
			r.Return("foo")
		}))

		client, _ := newTestPair(mux)
		defer client.Close()

		_, err := client.Call(ctx, "foo", nil, nil)
		fatal(t, err)

		mux.Remove("foo")

		_, err = client.Call(ctx, "foo", nil, nil)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("bad handler: nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("did not panic from nil handler")
			}
		}()
		mux := NewRespondMux()
		mux.Handle("foo.bar", nil)
	})

	t.Run("bad handle: exists", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("did not panic from existing handle")
			}
		}()
		mux := NewRespondMux()
		mux.Handle("foo", NotFoundHandler())
		mux.Handle("foo", NotFoundHandler())
	})
}

func TestRPC(t *testing.T) {
	ctx := context.Background()

	t.Run("unary rpc", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			var in string
			fatal(t, c.Receive(&in))
			r.Return(in)
		}))
		defer client.Close()

		var out string
		resp, err := client.Call(ctx, "", "Hello world", &out)
		fatal(t, err)
		if resp.Continue {
			t.Fatal("unexpected continue")
		}
		if out != "Hello world" {
			t.Fatalf("unexpected return: %#v", out)
		}
	})

	t.Run("unary rpc remote error", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			var in interface{}
			fatal(t, c.Receive(&in))
			r.Return(fmt.Errorf("internal server error"))
		}))
		defer client.Close()

		var out string
		_, err := client.Call(ctx, "", "Hello world", &out)
		if err == nil {
			t.Fatal("expected error")
		}
		if err != nil {
			rErr, ok := err.(RemoteError)
			if !ok {
				t.Fatal("unexpected error:", err)
			}
			if rErr.Error() != "remote: internal server error" {
				t.Fatal("unexpected error:", rErr)
			}
		}
	})

	t.Run("multi-return rpc", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			var in string
			fatal(t, c.Receive(&in))
			r.Return(in, strings.ToUpper(in))
		}))
		defer client.Close()

		var out, out2 string
		resp, err := client.Call(ctx, "", "Hello world", &out, &out2)
		fatal(t, err)
		if resp.Continue {
			t.Fatal("unexpected continue")
		}
		if out != "Hello world" {
			t.Errorf("unexpected return 1: %#v", out)
		}
		if out2 != "HELLO WORLD" {
			t.Errorf("unexpected return 2: %#v", out)
		}
	})

	t.Run("server streaming rpc", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			var in string
			fatal(t, c.Receive(&in))
			_, err := r.Continue(nil)
			fatal(t, err)
			fatal(t, r.Send(in))
			fatal(t, r.Send(in))
			fatal(t, r.Send(in))
		}))
		defer client.Close()

		resp, err := client.Call(ctx, "", "Hello world", nil)
		fatal(t, err)
		if !resp.Continue {
			t.Fatal("expected continue")
		}
		for i := 0; i < 3; i++ {
			var rcv string
			fatal(t, resp.Receive(&rcv))
			if rcv != "Hello world" {
				t.Fatalf("unexpected receive [%d]: %#v", i, rcv)
			}
		}

	})

	t.Run("client streaming rpc", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			for i := 0; i < 3; i++ {
				var rcv string
				fatal(t, c.Receive(&rcv))
				if rcv != "Hello world" {
					t.Fatalf("unexpected server receive [%d]: %#v", i, rcv)
				}
			}
		}))
		defer client.Close()

		sender := make(chan interface{})
		go func() {
			for i := 0; i < 3; i++ {
				sender <- "Hello world"
			}
			close(sender)
		}()
		_, err := client.Call(ctx, "", sender, nil)
		fatal(t, err)

	})

	t.Run("bidirectional streaming rpc", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			var rcv string
			for i := 0; i < 3; i++ {
				fatal(t, c.Receive(&rcv))
				if rcv != "Hello world" {
					t.Fatalf("unexpected server receive [%d]: %#v", i, rcv)
				}
			}
			_, err := r.Continue(nil)
			fatal(t, err)
			fatal(t, r.Send(rcv))
			fatal(t, r.Send(rcv))
			fatal(t, r.Send(rcv))
		}))
		defer client.Close()

		sender := make(chan interface{})
		go func() {
			for i := 0; i < 3; i++ {
				sender <- "Hello world"
			}
			close(sender)
		}()
		resp, err := client.Call(ctx, "", sender, nil)
		fatal(t, err)
		if !resp.Continue {
			t.Fatal("expected continue")
		}
		for i := 0; i < 3; i++ {
			var rcv string
			fatal(t, resp.Receive(&rcv))
			if rcv != "Hello world" {
				t.Fatalf("unexpected client receive [%d]: %#v", i, rcv)
			}
		}
	})

	t.Run("bidirectional channel byte stream", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			fatal(t, c.Receive(nil))
			ch, err := r.Continue(nil)
			fatal(t, err)
			io.Copy(ch, ch)
			ch.Close()
		}))
		defer client.Close()

		resp, err := client.Call(ctx, "", nil, nil)
		fatal(t, err)
		if !resp.Continue {
			t.Fatal("expected continue")
		}
		_, err = io.WriteString(resp.Channel, "Hello world")
		fatal(t, err)
		fatal(t, resp.Channel.CloseWrite())
		b, err := ioutil.ReadAll(resp.Channel)
		fatal(t, err)
		if string(b) != "Hello world" {
			t.Fatalf("unexpected data: %#v", b)
		}
	})

	t.Run("bidirectional channel codec stream", func(t *testing.T) {
		client, _ := newTestPair(HandlerFunc(func(r Responder, c *Call) {
			fatal(t, c.Receive(nil))
			_, err := r.Continue(nil)
			fatal(t, err)

			var rcv string
			for i := 0; i < 3; i++ {
				fatal(t, c.Receive(&rcv))
				if rcv != "Hello world" {
					t.Fatalf("unexpected server receive [%d]: %#v", i, rcv)
				}
			}
			fatal(t, r.Send(rcv))
			fatal(t, r.Send(rcv))
			fatal(t, r.Send(rcv))
		}))
		defer client.Close()

		resp, err := client.Call(ctx, "", nil, nil)
		fatal(t, err)
		if !resp.Continue {
			t.Fatal("expected continue")
		}
		fatal(t, resp.Send("Hello world"))
		fatal(t, resp.Send("Hello world"))
		fatal(t, resp.Send("Hello world"))
		for i := 0; i < 3; i++ {
			var rcv string
			fatal(t, resp.Receive(&rcv))
			if rcv != "Hello world" {
				t.Fatalf("unexpected client receive [%d]: %#v", i, rcv)
			}
		}
	})

}
