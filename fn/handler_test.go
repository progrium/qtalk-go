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

func (m *mockMethods) Struct(input *mockStruct) *mockStruct {
	return &mockStruct{strings.ToUpper(input.S)}
}

type mockStruct struct {
	S string
}

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
	h, _ = mux.Match("Struct")
	if h == nil {
		t.Fatal("expected Struct handler")
	}

	client, _ := rpctest.NewPair(mux, codec.JSONCodec{})
	defer client.Close()

	var ret string
	ctx := context.Background()
	if _, err := client.Call(ctx, "Foo", nil, &ret); err != nil {
		t.Error(err)
	}
	if ret != "Foo" {
		t.Errorf("unexpected ret: %v", ret)
	}

	var reply mockStruct
	if _, err := client.Call(ctx, "Struct", &mockStruct{S: "a"}, &reply); err != nil {
		t.Error(err)
	}
	if reply.S != "A" {
		t.Errorf("unexpected ret: %+v", reply)
	}
}
