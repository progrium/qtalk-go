package fn

import (
	"reflect"

	"github.com/progrium/qtalk-go/rpc"
)

func HandlerFrom(v interface{}) rpc.Handler {
	rv := reflect.Indirect(reflect.ValueOf(v))
	switch rv.Type().Kind() {
	case reflect.Func:
		return fromFunc(v, nil)
	case reflect.Struct:
		return fromMethods(v)
	default:
		panic("must be func or struct")
	}
}

func fromMethods(rcvr interface{}) rpc.Handler {
	t := reflect.TypeOf(rcvr)
	mux := rpc.NewRespondMux()
	for i := 0; i < t.NumMethod(); i++ {
		mux.Handle(t.Method(i).Name, fromFunc(t.Method(i).Func.Interface(), rcvr))
	}
	return mux
}

func fromFunc(fn_ interface{}, rcvr_ interface{}) rpc.Handler {
	fn := reflect.ValueOf(fn_)
	rcvr := reflect.ValueOf(rcvr_)
	fntyp := reflect.TypeOf(fn_)

	return rpc.HandlerFunc(func(r rpc.Responder, c *rpc.Call) {
		params := reflect.New(reflect.TypeOf([]interface{}{}))

		if err := c.Receive(params.Interface()); err != nil {
			// TODO: handle right, return error
			panic(err)
		}

		var fnParams []reflect.Value
		for idx, param := range params.Interface().([]interface{}) {
			if rcvr.IsValid() {
				idx++
			}
			switch fntyp.In(idx).Kind() {
			case reflect.Int:
				fnParams = append(fnParams, reflect.ValueOf(int(param.(float64))))
			default:
				fnParams = append(fnParams, reflect.ValueOf(param))
			}
		}

		// TODO capture panic: Call with too few input arguments
		// TODO type assertions for simple named types
		fnReturn := fn.Call(fnParams)

		r.Return(parseReturn(fnReturn))
	})
}

func parseReturn(ret []reflect.Value) interface{} {
	if len(ret) == 0 {
		return nil
	}

	var retVal reflect.Value
	errorInterface := reflect.TypeOf((*error)(nil)).Elem()

	// assuming up to 2 return values, one being an error
	for _, v := range ret[:2] {
		if v.Type().Implements(errorInterface) {
			if !v.IsNil() {
				return v.Interface().(error)
			}
		} else {
			retVal = v
		}
	}

	if !retVal.IsValid() {
		return nil
	}

	return retVal.Interface()
}
