package rpc

import (
	"fmt"
	"reflect"
)

func MustExport(v interface{}) map[string]Handler {
	h, err := Export(v)
	if err != nil {
		panic(err)
	}
	return h
}

func Export(v interface{}) (map[string]Handler, error) {
	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Func {
		h, e := exportFunc(v, nil)
		return map[string]Handler{"": h}, e
	}
	return exportStruct(rt, v)
}

func exportStruct(t reflect.Type, rcvr interface{}) (map[string]Handler, error) {
	handlers := make(map[string]Handler)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		handler, err := exportFunc(method.Func.Interface(), rcvr)
		if err != nil {
			return nil, fmt.Errorf("unable to export method %s: %s", method.Name, err.Error())
		}
		handlers[method.Name] = handler
	}
	return handlers, nil
}

func exportFunc(fn interface{}, rcvr interface{}) (Handler, error) {
	rfn := reflect.ValueOf(fn)
	rt := reflect.TypeOf(fn)

	if rt.Kind() != reflect.Func {
		return nil, fmt.Errorf("takes only a function")
	}

	var baseParams []reflect.Value
	var hasReceiver bool
	if rcvr != nil {
		if rt.NumIn() == 0 {
			return nil, fmt.Errorf("expecting 1 receiver argument, got 0")
		}
		hasReceiver = true
		baseParams = append(baseParams, reflect.ValueOf(rcvr))
	}

	if rt.NumOut() > 2 {
		return nil, fmt.Errorf("expecting 1 return value and optional error, got >2")
	}

	var pt reflect.Type
	if rt.NumIn() > len(baseParams)+1 {
		pt = reflect.TypeOf([]interface{}{})
	}
	if rt.NumIn() == len(baseParams)+1 {
		pt = rt.In(len(baseParams))
	}

	errorInterface := reflect.TypeOf((*error)(nil)).Elem()

	return HandlerFunc(func(r Responder, c *Call) {
		var params []reflect.Value
		for _, p := range baseParams {
			params = append(params, p)
		}

		if pt != nil {
			var pv reflect.Value
			if pt.Kind() == reflect.Ptr {
				pv = reflect.New(pt.Elem())
			} else {
				pv = reflect.New(pt)
			}

			err := c.Receive(pv.Interface())
			if err != nil {
				var debug interface{}
				c.Receive(&debug)
				fmt.Println(debug)
				// arguments weren't what was expected,
				// or any other error
				panic(err)
			}

			switch pt.Kind() {
			case reflect.Slice:
				startIdx := len(params)
				args := reflect.Indirect(pv).Interface().([]interface{})
				for idx, arg := range args {
					if startIdx+idx >= rt.NumIn() {
						break
					}
					if rt.In(startIdx+idx).Kind() == reflect.Int {
						params = append(params, reflect.ValueOf(int(arg.(float64))))
					} else {
						params = append(params, reflect.ValueOf(arg))
					}
				}
				expected := rt.NumIn()
				if hasReceiver {
					expected--
				}
				if len(args) < expected {
					params = append(params, reflect.ValueOf(c))
				}
			case reflect.Ptr:
				params = append(params, pv)
			default:
				params = append(params, pv.Elem())
			}
		}

		// TODO capture panic: Call with too few input arguments
		// TODO type assertions for simple named types
		retVals := rfn.Call(params)

		if len(retVals) == 0 {
			r.Return(nil)
			return
		}

		// assuming up to 2 return values, one being an error
		var retVal reflect.Value
		for _, v := range retVals {
			if v.Type().Implements(errorInterface) {
				if !v.IsNil() {
					r.Return(v.Interface().(error))
					return
				}
			} else {
				retVal = v
			}
		}

		if !retVal.IsValid() {
			r.Return(nil)
		} else {
			r.Return(retVal.Interface())
		}

	}), nil
}
