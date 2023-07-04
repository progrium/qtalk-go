package libp2p_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"

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

func listener(t *testing.T) (libp2p.Conn, string) {
	t.Helper()
	token := generateToken()
	l, err := libp2p.Listen(token)
	fatal(err, t)
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	return l, token
}

func TestSingleChannelEcho(t *testing.T) {
	l, token := listener(t)
	SingleChannelEcho(t, token, l.Accept, libp2p.Dial)
}

func TestMultiChannelEcho(t *testing.T) {
	l, token := listener(t)
	MultiChannelEcho(t, token, l.Accept, libp2p.Dial)
}

func TestOpenTimeout(t *testing.T) {
	t.Skipf("This test should detect that Open will time out if the remote side does not call Accept. However, that is not implemented yet.")

	l, addr := listener(t)
	OpenTimeout(t, addr, l.Accept, libp2p.Dial)
}

func listener2(t *testing.T) (AcceptFunc, string) {
	t.Helper()
	token := generateToken()
	l, err := libp2p.Listen2(context.Background(), token)
	fatal(err, t)
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	return l.Accept, token
}

func TestSingleChannelEcho2(t *testing.T) {
	accept, token := listener2(t)
	SingleChannelEcho(t, token, accept, libp2p.Dial2)
}

func TestMultiChannelEcho2(t *testing.T) {
	accept, token := listener2(t)
	MultiChannelEcho(t, token, accept, libp2p.Dial2)
}

func TestOpenTimeout2(t *testing.T) {
	t.Skipf("This test should detect that Open will time out if the remote side does not call Accept. However, that is not implemented yet.")

	accept, token := listener2(t)
	OpenTimeout(t, token, accept, libp2p.Dial2)
}
