package exp_test

import (
	"context"
	"encoding/json"
	"testing"

	fn "github.com/progrium/qtalk-go/exp"
	"github.com/progrium/qtalk-go/rpc"
)

type mockCaller struct {
	selector      string
	params, reply interface{}
}

func (mc *mockCaller) Call(ctx context.Context, selector string, params, reply interface{}) (*rpc.Response, error) {
	mc.selector = selector
	mc.params = params
	mc.reply = reply
	return nil, nil
}

type mockData struct {
	Inner struct {
		Fn *fn.Ptr
	}
	NilFn *fn.Ptr
	Fn    *fn.Ptr
}

func TestPtrCall(t *testing.T) {
	cb := fn.Callback(func() {})
	data := mockData{Fn: cb}
	caller := &mockCaller{}
	fn.SetCallers(&data, caller)
	var ret interface{}
	cb.Call(context.Background(), []int{1, 2, 3}, &ret)
	if len(caller.params.([]int)) != 3 {
		t.Fatal("unexpected params:", caller.params)
	}
	if cb.Ptr != caller.selector {
		t.Fatal("unexpected selector:", caller.selector)
	}
}

func TestPtrsFromMap(t *testing.T) {
	data := mockData{Fn: fn.Callback(func() {})}

	b, _ := json.Marshal(data)
	var m map[string]interface{}
	json.Unmarshal(b, &m)

	caller := &mockCaller{}
	fn.SetCallers(&m, caller)

	if m["Fn"].(map[string]interface{})["Caller"] != caller {
		t.Fatal("caller not set")
	}

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
