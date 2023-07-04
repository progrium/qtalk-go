package libp2p_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/talk"
)

type AcceptFunc func() (mux.Session, error)

func SingleChannelEcho(t *testing.T, addr string, accept AcceptFunc, dial talk.Dialer) {
	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		sess, err := accept()
		fatal(err, t)

		ch, err := sess.Open(context.Background())
		fatal(err, t)
		b, err := ioutil.ReadAll(ch)
		fatal(err, t)
		_, err = ch.Write(b)
		fatal(err, t)
		err = ch.CloseWrite()
		ch.Close() // should already be closed by other end

		<-testComplete
		err = sess.Close()
		fatal(err, t)
		close(sessionClosed)
	}()

	sess, err := dial(addr)
	fatal(err, t)

	var ch mux.Channel
	if !t.Run("session accept", func(t *testing.T) {
		ch, err = sess.Accept()
		fatal(err, t)
	}) {
		return
	}

	if !t.Run("channel write", func(t *testing.T) {
		_, err = ch.Write([]byte("Hello world"))
		fatal(err, t)
		err = ch.CloseWrite()
		fatal(err, t)
	}) {
		return
	}

	var b []byte
	if !t.Run("channel read", func(t *testing.T) {
		b, err = ioutil.ReadAll(ch)
		fatal(err, t)
		ch.Close() // should already be closed by other end
	}) {
		return
	}

	if !bytes.Equal(b, []byte("Hello world")) {
		t.Fatalf("unexpected bytes: %s", b)
	}
	close(testComplete)
	<-sessionClosed
}

func MultiChannelEcho(t *testing.T, addr string, accept AcceptFunc, dial talk.Dialer) {
	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		sess, err := accept()
		fatal(err, t)

		ch, err := sess.Open(context.Background())
		fatal(err, t)
		b, err := ioutil.ReadAll(ch)
		fatal(err, t)
		ch.Close() // should already be closed by other end
		if string(b) != "Hello world" {
			t.Errorf("got: %#v", b)
		}

		ch, err = sess.Accept()
		fatal(err, t)
		_, err = ch.Write(b)
		fatal(err, t)
		err = ch.CloseWrite()
		fatal(err, t)

		<-testComplete
		err = sess.Close()
		fatal(err, t)
		close(sessionClosed)
	}()

	sess, err := dial(addr)
	fatal(err, t)

	var ch mux.Channel
	if !t.Run("session accept", func(t *testing.T) {
		ch, err = sess.Accept()
		fatal(err, t)
	}) {
		return
	}

	if !t.Run("channel write", func(t *testing.T) {
		_, err = ch.Write([]byte("Hello world"))
		fatal(err, t)
		err = ch.Close()
		fatal(err, t)
	}) {
		return
	}

	if !t.Run("session open", func(t *testing.T) {
		ch, err = sess.Open(context.Background())
		fatal(err, t)
	}) {
		return
	}

	var b []byte
	if !t.Run("channel read", func(t *testing.T) {
		b, err = ioutil.ReadAll(ch)
		fatal(err, t)
		ch.Close() // should already be closed by other end
	}) {
		return
	}

	if !bytes.Equal(b, []byte("Hello world")) {
		t.Fatalf("unexpected bytes: %s", b)
	}
	close(testComplete)
	<-sessionClosed
}

func OpenTimeout(t *testing.T, addr string, accept AcceptFunc, dial talk.Dialer) {
	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		sess, err := accept()
		fatal(err, t)

		<-testComplete
		err = sess.Close()
		fatal(err, t)
		close(sessionClosed)
	}()

	sess, err := dial(addr)
	fatal(err, t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = sess.Open(ctx)
	if err == nil {
		t.Fatalf("expected Open to time out")
	}

	close(testComplete)
	<-sessionClosed
}
