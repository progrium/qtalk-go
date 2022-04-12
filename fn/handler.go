package fn

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/progrium/qtalk-go/rpc"
)

// HandlerFrom uses reflection to return a handler from either a function or
// methods from a struct. When a struct is used, HandlerFrom creates a RespondMux
// registering each method as a handler using its method name. From there, methods
// are treated just like functions.
//
// Function handlers expect an array to use as arguments. If the incoming argument
// array is too large or too small, the handler returns an error. Functions can opt-in
// to take a final Call pointer argument, allowing the handler to give it the Call value
// being processed. Functions can return nothing which the handler returns as nil, or
// a single value which can be an error, or two values where one value is an error.
// In the latter case, the value is returned if the error is nil, otherwise just the
// error is returned. Handlers based on functions that return more than two values will
// simply ignore the remaining values.
//
// Structs that implement the Handler interface will be added as a catch-all handler
// along with their individual methods. This lets you implement dynamic methods.
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

// Args is the expected argument value for calls made to HandlerFrom handlers.
// Since it is just a slice of empty interface values, you can alternatively use
// more specific slice types ([]int{}, etc) if all arguments are of the same type.
type Args []interface{}

func fromMethods(rcvr interface{}) rpc.Handler {
	t := reflect.TypeOf(rcvr)
	mux := rpc.NewRespondMux()
	for i := 0; i < t.NumMethod(); i++ {
		mux.Handle(t.Method(i).Name, fromFunc(t.Method(i).Func.Interface(), rcvr))
	}
	h, ok := rcvr.(rpc.Handler)
	if ok {
		mux.Handle("/", h)
	}
	return mux
}

func fromFunc(fn_ interface{}, rcvr_ interface{}) rpc.Handler {
	fn := reflect.ValueOf(fn_)
	rcvr := reflect.ValueOf(rcvr_)
	fntyp := reflect.TypeOf(fn_)

	return rpc.HandlerFunc(func(r rpc.Responder, c *rpc.Call) {
		defer func() {
			if p := recover(); p != nil {
				r.Return(fmt.Errorf("panic: %s", p))
			}
		}()

		params := reflect.New(reflect.TypeOf([]interface{}{}))

		if err := c.Receive(params.Interface()); err != nil {
			r.Return(fmt.Errorf("fn: args: %s", err.Error()))
			return
		}

		if params.Elem().Len() > fn.Type().NumIn() {
			r.Return(errors.New("fn: too many input arguments"))
			return
		}

		var fnParams []reflect.Value

		if rcvr.IsValid() {
			fnParams = append(fnParams, rcvr)
		}

		for idx, param := range params.Elem().Interface().([]interface{}) {
			if rcvr.IsValid() {
				idx++
			}
			switch fntyp.In(idx).Kind() {
			case reflect.Struct:
				// decode to struct type using mapstructure
				arg := reflect.New(fntyp.In(idx))
				if err := mapstructure.Decode(param, arg.Interface()); err != nil {
					r.Return(fmt.Errorf("fn: mapstructure: %s", err.Error()))
					return
				}
				fnParams = append(fnParams, ensureType(arg.Elem(), fntyp.In(idx)))
			case reflect.Slice:
				rv := reflect.ValueOf(param)
				// decode slice of structs to struct type using mapstructure
				if fntyp.In(idx).Elem().Kind() == reflect.Struct {
					nv := reflect.MakeSlice(fntyp.In(idx), rv.Len(), rv.Len())
					for i := 0; i < rv.Len(); i++ {
						ref := reflect.New(nv.Index(i).Type())
						if err := mapstructure.Decode(rv.Index(i).Interface(), ref.Interface()); err != nil {
							r.Return(fmt.Errorf("fn: mapstructure: %s", err.Error()))
							return
						}
						nv.Index(i).Set(reflect.Indirect(ref))
					}
					rv = nv
				}
				fnParams = append(fnParams, rv)
			case reflect.Int:
				// if int is expected cast the float64 (assumes json-like encoding)
				fnParams = append(fnParams, ensureType(reflect.ValueOf(int(param.(float64))), fntyp.In(idx)))
			default:
				fnParams = append(fnParams, ensureType(reflect.ValueOf(param), fntyp.In(idx)))
			}
		}

		// if the last argument in fn is an rpc.Call, add our call to fnParams
		if fn.Type().NumIn() > 0 {
			callRef := reflect.TypeOf(&rpc.Call{})
			if fn.Type().In(fn.Type().NumIn()-1) == callRef {
				fnParams = append(fnParams, reflect.ValueOf(c))
			}
		}

		if len(fnParams) < fn.Type().NumIn() {
			r.Return(errors.New("fn: too few input arguments"))
			return
		}

		fnReturn := fn.Call(fnParams)

		r.Return(parseReturn(fnReturn))
	})
}

// parseReturn turns a slice of reflect.Values into a value or an error
func parseReturn(ret []reflect.Value) interface{} {
	if len(ret) == 0 {
		return nil
	}
	if len(ret) == 1 {
		return ret[0].Interface()
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

	return retVal.Interface()
}

// ensureType ensures a value is converted to the expected
// defined type from a convertable underlying type
func ensureType(v reflect.Value, t reflect.Type) reflect.Value {
	nv := v
	if v.Type().Kind() == reflect.Slice && v.Type().Elem() != t {
		switch t.Kind() {
		case reflect.Array:
			nv = reflect.Indirect(reflect.New(t))
			for i := 0; i < v.Len(); i++ {
				vv := reflect.ValueOf(v.Index(i).Interface())
				nv.Index(i).Set(vv.Convert(nv.Type().Elem()))
			}
		case reflect.Slice:
			nv = reflect.MakeSlice(t, 0, 0)
			for i := 0; i < v.Len(); i++ {
				vv := reflect.ValueOf(v.Index(i).Interface())
				nv = reflect.Append(nv, vv.Convert(nv.Type().Elem()))
			}
		default:
			panic("unable to convert slice to non-array, non-slice type")
		}
	}
	if v.Type() != t {
		nv = nv.Convert(t)
	}
	return nv
}
