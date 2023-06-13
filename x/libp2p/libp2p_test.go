package libp2p_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"testing"
	"time"

	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/x/libp2p"
)

func fatal(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func generateToken() string {
	var b [100]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b[:])
}

func TestSingleChannelEcho(t *testing.T) {
	token := generateToken()
	l, err := libp2p.Listen(token)
	fatal(err, t)
	defer l.Close()

	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		sess, err := l.Accept()
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

	sess, err := libp2p.Dial(token)
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

func TestMultiChannelEcho(t *testing.T) {
	token := generateToken()
	l, err := libp2p.Listen(token)
	fatal(err, t)
	defer l.Close()

	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		sess, err := l.Accept()
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

	sess, err := libp2p.Dial(token)
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

func TestOpenTimeout(t *testing.T) {
	t.Skipf("This test should detect that Open will time out if the remote side does not call Accept. However, that is not implemented yet.")

	token := generateToken()
	l, err := libp2p.Listen(token)
	fatal(err, t)
	defer l.Close()

	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		sess, err := l.Accept()
		fatal(err, t)

		<-testComplete
		err = sess.Close()
		fatal(err, t)
		close(sessionClosed)
	}()

	sess, err := libp2p.Dial(token)
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
