package mux

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

func init() {
	openTimeout = 100 * time.Millisecond
}

func fatal(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestQmux(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err, t)
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		fatal(err, t)
		defer conn.Close()

		sess := New(conn)

		ch, err := sess.Open(context.Background())
		fatal(err, t)
		b, err := ioutil.ReadAll(ch)
		fatal(err, t)
		ch.Close() // should already be closed by other end

		ch, err = sess.Accept()
		_, err = ch.Write(b)
		fatal(err, t)
		err = ch.CloseWrite()
		fatal(err, t)

		err = sess.Close()
		fatal(err, t)
	}()

	conn, err := net.Dial("tcp", l.Addr().String())
	fatal(err, t)
	defer conn.Close()

	sess := New(conn)

	var ch *Channel
	t.Run("session accept", func(t *testing.T) {
		ch, err = sess.Accept()
		fatal(err, t)
	})

	t.Run("channel write", func(t *testing.T) {
		_, err = ch.Write([]byte("Hello world"))
		fatal(err, t)
		err = ch.Close()
		fatal(err, t)
	})

	t.Run("session open", func(t *testing.T) {
		ch, err = sess.Open(context.Background())
		fatal(err, t)
	})

	var b []byte
	t.Run("channel read", func(t *testing.T) {
		b, err = ioutil.ReadAll(ch)
		fatal(err, t)
		ch.Close() // should already be closed by other end
	})

	if !bytes.Equal(b, []byte("Hello world")) {
		t.Fatalf("unexpected bytes: %s", b)
	}
}

func TestSessionOpenClientTimeout(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err, t)
	defer l.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	fatal(err, t)
	defer conn.Close()

	sess := New(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	ch, err := sess.Open(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, but got: %v", err)
	}
	if ch != nil {
		ch.Close()
	}
}

func TestSessionOpenServerTimeout(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err, t)
	defer l.Close()

	errCh := make(chan error)
	go func() {
		conn, err := net.Dial("tcp", l.Addr().String())
		fatal(err, t)
		defer conn.Close()

		sess := New(conn)
		defer sess.Close()

		_, err = sess.Open(context.Background())
		errCh <- err
	}()

	conn, err := l.Accept()
	fatal(err, t)
	defer conn.Close()

	sess := New(conn)
	defer sess.Close()

	if <-errCh == nil {
		t.Errorf("expected open to fail when listener doesn't call Accept")
	}
	fatal(sess.Close(), t)
}

func TestSessionWait(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err, t)
	defer l.Close()

	conn, err := net.Dial("tcp", l.Addr().String())
	fatal(err, t)
	defer conn.Close()

	sess := New(conn)
	fatal(sess.Close(), t)
	// wait should return immediately since the connection was closed
	err = sess.Wait()
	var netErr net.Error
	if !errors.As(err, &netErr) {
		t.Fatalf("expected a network error, but got: %v", err)
	}
}
