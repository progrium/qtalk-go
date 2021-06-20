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

func TestHandlerFromFunc(t *testing.T) {
	t.Run("int sum", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) int {
			return a + b
		}), codec.JSONCodec{})

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

		if _, err := client.Call(context.Background(), "", []interface{}{2, 3}, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not enough args", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int) int {
			return a + b
		}), codec.JSONCodec{})

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

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2, 3, 5}, &sum)
		if err == nil || !strings.Contains(err.Error(), "too many") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("with call", func(t *testing.T) {
		client, _ := rpctest.NewPair(HandlerFrom(func(a, b int, call *rpc.Call) int {
			if call.Selector != "sum" {
				t.Fatalf("unexpected selector: %v", call.Selector)
			}
			return a + b
		}), codec.JSONCodec{})

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

		var sum int
		_, err := client.Call(context.Background(), "", []interface{}{2, 3}, &sum)
		if err == nil || !strings.Contains(err.Error(), "test") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

}
