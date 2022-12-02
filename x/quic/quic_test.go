package quic

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/progrium/qtalk-go/mux"
)

func fatal(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}

func TestOpenTimeout(t *testing.T) {
	t.Skipf("broken")

	l, err := quic.ListenAddr("127.0.0.1:0", generateTLSConfig(), nil)
	fatal(err, t)
	defer l.Close()

	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		conn, err := l.Accept(context.Background())
		fatal(err, t)
		// defer conn.Close()

		sess := New(conn)

		<-testComplete
		err = sess.Close()
		fatal(err, t)
		close(sessionClosed)
	}()

	addr := l.Addr().String()
	conn, err := quic.DialAddr(addr, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}, nil)
	fatal(err, t)
	// defer conn.Close()

	sess := New(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = sess.Open(ctx)
	if err == nil {
		t.Fatalf("expected Open to time out")
	}

	close(testComplete)
	<-sessionClosed
}

func TestSingleChannelEcho(t *testing.T) {
	l, err := quic.ListenAddr("127.0.0.1:0", generateTLSConfig(), nil)
	fatal(err, t)
	defer l.Close()

	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		conn, err := l.Accept(context.Background())
		fatal(err, t)
		// defer conn.Close()

		sess := New(conn)

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

	addr := l.Addr().String()
	conn, err := quic.DialAddr(addr, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}, nil)
	fatal(err, t)
	// defer conn.Close()

	sess := New(conn)

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
	l, err := quic.ListenAddr("127.0.0.1:0", generateTLSConfig(), nil)
	fatal(err, t)
	defer l.Close()

	testComplete := make(chan struct{})
	sessionClosed := make(chan struct{})

	go func() {
		conn, err := l.Accept(context.Background())
		fatal(err, t)
		// defer conn.Close()

		sess := New(conn)

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

	addr := l.Addr().String()
	conn, err := quic.DialAddr(addr, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}, nil)
	fatal(err, t)
	// defer conn.Close()

	sess := New(conn)

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
