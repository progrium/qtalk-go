package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"
	"os"

	"github.com/progrium/qtalk-go/cmd/qtalk/cli"
	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/fn"
	"github.com/progrium/qtalk-go/interop"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/rpc"
	cbor "github.com/progrium/qtalk-go/x/cbor/codec"
	qquic "github.com/progrium/qtalk-go/x/quic"
	"github.com/quic-go/quic-go"
)

var interopCmd = &cli.Command{
	Usage: "interop",
	Short: "run interop service",
	Args:  cli.MaxArgs(1),
	Run: func(ctx context.Context, args []string) {
		log.SetOutput(os.Stderr)

		var c codec.Codec = cbor.CBORCodec{}
		if os.Getenv("QTALK_CODEC") == "json" {
			c = codec.JSONCodec{}
		}

		if len(args) == 0 {
			// STDIO
			sess, err := mux.DialStdio()
			fatal(err)
			serve(sess, c)
			return
		}

		// QUIC
		log.Printf("* Listening on %s...\n", args[0])
		l, err := quic.ListenAddr(args[0], generateTLSConfig(), nil)
		fatal(err)
		defer l.Close()

		for {
			conn, err := l.Accept(context.Background())
			fatal(err)
			go serve(qquic.New(conn), c)
		}
	},
}

func serve(sess mux.Session, c codec.Codec) {
	srv := rpc.Server{
		Handler: fn.HandlerFrom(interop.InteropService{}),
		Codec:   c,
	}
	srv.Respond(sess, nil)
}

var defaultTLSConfig = tls.Config{
	NextProtos: []string{"qtalk-quic"},
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
	cfg := defaultTLSConfig.Clone()
	cfg.Certificates = []tls.Certificate{tlsCert}
	return cfg
}
