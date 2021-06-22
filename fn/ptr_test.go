package fn_test

import (
	"context"
	"testing"

	"github.com/progrium/qtalk-go/fn"
	"github.com/progrium/qtalk-go/rpc"
)

type mockCaller struct{}

func (mc *mockCaller) Call(ctx context.Context, selector string, params, reply interface{}) (*rpc.Response, error) {
	return nil, nil
}

type mockData struct {
	Inner struct {
		Fn *fn.Ptr
	}
	NilFn *fn.Ptr
	Fn    *fn.Ptr
}

func TestCallbackUtils(t *testing.T) {
	data := mockData{
		Fn: fn.Callback(func() {
			//outerCalled = true
		}),
	}
	data.Inner.Fn = fn.Callback(func() {
		//innerCalled = true
	})

	caller := &mockCaller{}
	fn.SetCallers(&data, caller)

	t.Run("SetCallers", func(t *testing.T) {
		if data.Fn.Caller != caller {
			t.Fatal("outer caller not set")
		}

		if data.Inner.Fn.Caller != caller {
			t.Fatal("inner caller not set")
		}
	})

	mux := rpc.NewRespondMux()
	fn.RegisterPtrs(mux, data)

	t.Run("RegisterPtrs", func(t *testing.T) {
		h, _ := mux.Match(data.Fn.Ptr)
		if h == nil {
			t.Fatal("outer handler not found")
		}

		h, _ = mux.Match(data.Inner.Fn.Ptr)
		if h == nil {
			t.Fatal("inner handler not found")
		}
	})

	fn.UnregisterPtrs(mux, data)

	t.Run("UnregisterPtrs", func(t *testing.T) {
		h, _ := mux.Match(data.Fn.Ptr)
		if h != nil {
			t.Fatal("outer handler still found")
		}

		h, _ = mux.Match(data.Inner.Fn.Ptr)
		if h != nil {
			t.Fatal("inner handler still found")
		}
	})

}
