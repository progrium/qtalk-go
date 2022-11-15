package fn

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/progrium/qtalk-go/rpc/rpctest"
)

func TestHandlerFromBadData(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("did not panic from bad argument data")
		}
	}()
	HandlerFrom(2)
}

type subfake struct {
	A string
}

type fake struct {
	A subfake
	B int
}

type id int

func TestHandlerFromFunc(t *testing.T) {
	t.Run("int sum", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) int {
			return a + b
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		if _, err := client.Call(context.Background(), "", []interface{}{2, 3}, &sum); err != nil {
			t.Fatal(err)
		}
		if sum != 5 {
			t.Fatalf("unexpected sum: %v", sum)
		}
	})

	t.Run("defined type arg and return", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a id) id {
			return a
		}), codec.JSONCodec{})
		defer client.Close()

		var ret id
		if _, err := client.Call(context.Background(), "", Args{id(64)}, &ret); err != nil {
			t.Fatal(err)
		}
		if ret != 64 {
			t.Fatalf("unexpected return value: %v", ret)
		}
	})

	t.Run("struct arguments", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a fake, b subfake) {
			if a.A.A != "Hello" {
				t.Fatalf("unexpected field value in struct: %v", a)
			}
			if b.A != "world" {
				t.Fatalf("unexpected field value in struct: %v", b)
			}
		}), codec.JSONCodec{})
		defer client.Close()

		if _, err := client.Call(context.Background(), "", Args{fake{A: subfake{A: "Hello"}}, subfake{A: "world"}}, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("nil error", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) error {
			return nil
		}), codec.JSONCodec{})
		defer client.Close()

		if _, err := client.Call(context.Background(), "", []interface{}{2, 3}, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not enough args", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) int {
			return a + b
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2}, &sum)
		if err == nil || !strings.Contains(err.Error(), "too few") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("too many args", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) int {
			return a + b
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2, 3, 5}, &sum)
		if err == nil || !strings.Contains(err.Error(), "too many") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("with call", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int, call *rpc.Call) int {
			if call.Selector != "/sum" {
				t.Fatalf("unexpected selector: %v", call.Selector)
			}
			return a + b
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		if _, err := client.Call(context.Background(), "sum", []interface{}{2, 3}, &sum); err != nil {
			t.Fatal(err)
		}
		if sum != 5 {
			t.Fatalf("unexpected sum: %v", sum)
		}
	})

	t.Run("return error", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) error {
			return errors.New("test")
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2, 3}, &sum)
		if err == nil || !strings.Contains(err.Error(), "test") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("return error with value", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) (int, error) {
			return a + b, errors.New("test")
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2, 3}, &sum)
		if err == nil || !strings.Contains(err.Error(), "test") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no return", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) {
			return
		}), codec.JSONCodec{})
		defer client.Close()

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2, 3}, &sum)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

}

type mockMethods struct{}

func (m *mockMethods) Foo() string {
	return "Foo"
}

func (m *mockMethods) Bar() {}

func TestHandlerFromMethods(t *testing.T) {
	handler := HandlerFrom(&mockMethods{})
	mux, ok := handler.(*rpc.RespondMux)
	if !ok {
		t.Fatal("expected handler to be rpc.RespondMux")
	}
	h, _ := mux.Match("Foo")
	if h == nil {
		t.Fatal("expected Foo handler")
	}
	h, _ = mux.Match("Bar")
	if h == nil {
		t.Fatal("expected Bar handler")
	}

	client, _ := rpctest.NewPair(mux, codec.JSONCodec{})
	defer client.Close()

	var ret string
	if _, err := client.Call(context.Background(), "Foo", nil, &ret); err != nil {
		t.Fatal(err)
	}
	if ret != "Foo" {
		t.Fatalf("unexpected ret: %v", ret)
	}
}

func TestHandlerFromMethodsInterface(t *testing.T) {
	handler := HandlerFrom[interface {
		Foo() string
	}](&mockMethods{})
	mux, ok := handler.(*rpc.RespondMux)
	if !ok {
		t.Fatal("expected handler to be rpc.RespondMux")
	}
	h, _ := mux.Match("Foo")
	if h == nil {
		t.Fatal("expected Foo handler")
	}
	h, _ = mux.Match("Bar")
	if h != nil {
		t.Fatal("expected no handler for Bar method not on interface")
	}

	client, _ := rpctest.NewPair(mux, codec.JSONCodec{})
	defer client.Close()

	var ret string
	if _, err := client.Call(context.Background(), "Foo", nil, &ret); err != nil {
		t.Fatal(err)
	}
	if ret != "Foo" {
		t.Fatalf("unexpected ret: %v", ret)
	}
}

func TestHandlerFromMethodsInterfaceDifferentMethod(t *testing.T) {
	// Also check a different method to ensure that the reflection code is
	// matching the correct method based on the interface and not just getting
	// Method(0) which matches up in the first test.
	handler := HandlerFrom[interface {
		Bar()
	}](&mockMethods{})
	mux, ok := handler.(*rpc.RespondMux)
	if !ok {
		t.Fatal("expected handler to be rpc.RespondMux")
	}
	h, _ := mux.Match("Bar")
	if h == nil {
		t.Fatal("expected Bar handler")
	}
	h, _ = mux.Match("Foo")
	if h != nil {
		t.Fatal("expected no handler for Foo method not on interface")
	}

	client, _ := rpctest.NewPair(mux, codec.JSONCodec{})
	defer client.Close()

	var ret string
	if _, err := client.Call(context.Background(), "Bar", nil, &ret); err != nil {
		t.Fatal(err)
	}
	if ret != "" {
		t.Fatalf("unexpected ret: %v", ret)
	}
}
