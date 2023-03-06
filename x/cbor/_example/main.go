package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/progrium/qtalk-go/fn"
	"github.com/progrium/qtalk-go/talk"
	"github.com/progrium/qtalk-go/x/cbor/codec"
	"github.com/progrium/qtalk-go/x/cbor/mux"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type Foo struct {
	S string
	F *Foo
}

type Foos []Foo

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	fatal(err)
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		fatal(err)
		sess := talk.NewPeer(mux.New(conn), codec.CBORCodec{})
		sess.Handle("hello", fn.HandlerFrom(func(s string, f float64, i int, b []byte) bool {
			log.Printf("%#v %#v %#v %#v", s, f, i, b)
			return true
		}))
		sess.Handle("foo", fn.HandlerFrom(func(f1 []Foo, f2 Foos) {
			log.Printf("%#v %#v", f1, f2)
		}))
		sess.Respond()
	}()

	conn, err := net.Dial("tcp", l.Addr().String())
	fatal(err)
	defer conn.Close()

	sess := talk.NewPeer(mux.New(conn), codec.CBORCodec{})
	var ret bool
	_, err = sess.Call(context.Background(), "hello", fn.Args{"hi", 1.23, 100, []byte("hello")}, &ret)
	fatal(err)
	fmt.Println(ret)

	foo := Foo{S: "Subfoo"}
	_, err = sess.Call(context.Background(), "foo", fn.Args{[]Foo{
		Foo{S: "Foo1"},
		Foo{S: "Foo2", F: &foo},
	}, Foos{
		Foo{S: "FooA"},
		Foo{S: "FooB"},
	}}, &ret)
	fatal(err)
}
