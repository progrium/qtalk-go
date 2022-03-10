package mux

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"syscall"
	"testing"
)

func setupProxy(t *testing.T) (func(), chan error, *Session, *Session) {
	la, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err, t)
	lb, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err, t)
	cleanup := func() {
		la.Close()
		lb.Close()
	}

	proxyErr := make(chan error, 1)
	go func() {
		a, err := la.Accept()
		fatal(err, t)
		defer a.Close()

		b, err := lb.Accept()
		fatal(err, t)
		defer b.Close()

		proxyErr <- Proxy(New(b), New(a))
	}()

	cb, err := net.Dial("tcp", lb.Addr().String())
	fatal(err, t)

	ca, err := net.Dial("tcp", la.Addr().String())
	fatal(err, t)

	return cleanup, proxyErr, New(ca), New(cb)
}

func TestProxyDuplex(t *testing.T) {
	cleanup, _, sessA, sessB := setupProxy(t)
	defer cleanup()

	ctx := context.Background()
	chA, err := sessA.Open(ctx)
	fatal(err, t)

	chB, err := sessB.Accept()
	fatal(err, t)

	msgA := "A -> a <-> b -> B"
	msgB := "B -> b <-> a -> A"
	_, err = io.WriteString(chA, msgA)
	fatal(err, t)
	fatal(chA.CloseWrite(), t)

	_, err = io.WriteString(chB, msgB)
	fatal(err, t)
	fatal(chB.CloseWrite(), t)

	gotA, err := ioutil.ReadAll(chA)
	fatal(err, t)
	gotB, err := ioutil.ReadAll(chB)
	fatal(err, t)

	if string(gotA) != msgB {
		t.Fatalf("unexpected bytes read from chA: %#v", gotA)
	}

	if string(gotB) != msgA {
		t.Fatalf("unexpected bytes read from chB: %#v", gotB)
	}
}

func TestProxyCloseDst(t *testing.T) {
	cleanup, proxyErr, sessA, sessB := setupProxy(t)
	defer cleanup()

	fatal(sessB.Close(), t)

	ctx := context.Background()
	chA, err := sessA.Open(ctx)
	fatal(err, t)

	_, err = io.WriteString(chA, "hello")
	if err != nil &&
		!errors.Is(err, net.ErrClosed) &&
		!errors.Is(err, io.EOF) &&
		!errors.Is(err, syscall.EPIPE) &&
		!errors.Is(err, syscall.ECONNRESET) {
		t.Fatal("unexpected channel error:", err)
	}

	err = <-proxyErr
	if !errors.Is(err, net.ErrClosed) {
		t.Fatal("unexpected proxy error:", err)
	}
}
