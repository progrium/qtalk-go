package fn

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

// Call wraps invoking a function via reflection, converting the arguments with
// ArgsTo and the returns with ParseReturn.
func Call(fn any, args []any) (_ []any, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %s [%s]", p, identifyPanic())
		}
	}()
	fnval := reflect.ValueOf(fn)
	fnParams, err := ArgsTo(fnval.Type(), args)
	if err != nil {
		return nil, err
	}
	fnReturn := fnval.Call(fnParams)
	return ParseReturn(fnReturn)
}

// ArgsTo converts the arguments into `reflect.Value`s suitable to pass as
// parameters to a function with the given type via reflection.
func ArgsTo(fntyp reflect.Type, args []any) ([]reflect.Value, error) {
	if len(args) != fntyp.NumIn() {
		return nil, fmt.Errorf("fn: expected %d params, got %d", fntyp.NumIn(), len(args))
	}
	fnParams := make([]reflect.Value, len(args))
	for idx, param := range args {
		switch fntyp.In(idx).Kind() {
		case reflect.Struct:
			// decode to struct type using mapstructure
			arg := reflect.New(fntyp.In(idx))
			if err := mapstructure.Decode(param, arg.Interface()); err != nil {
				return nil, fmt.Errorf("fn: mapstructure: %s", err.Error())
			}
			fnParams[idx] = ensureType(arg.Elem(), fntyp.In(idx))
		case reflect.Slice:
			rv := reflect.ValueOf(param)
			// decode slice of structs to struct type using mapstructure
			if fntyp.In(idx).Elem().Kind() == reflect.Struct {
				nv := reflect.MakeSlice(fntyp.In(idx), rv.Len(), rv.Len())
				for i := 0; i < rv.Len(); i++ {
					ref := reflect.New(nv.Index(i).Type())
					if err := mapstructure.Decode(rv.Index(i).Interface(), ref.Interface()); err != nil {
						return nil, fmt.Errorf("fn: mapstructure: %s", err.Error())
					}
					nv.Index(i).Set(reflect.Indirect(ref))
				}
				rv = nv
			}
			fnParams[idx] = rv
		case reflect.Int:
			// if int is expected cast the float64 (assumes json-like encoding)
			fnParams[idx] = ensureType(reflect.ValueOf(int(param.(float64))), fntyp.In(idx))
		default:
			fnParams[idx] = ensureType(reflect.ValueOf(param), fntyp.In(idx))
		}
	}
	return fnParams, nil
}

// ParseReturn splits the results of reflect.Call() into the values, and
// possibly an error.
// If the last value is a non-nil error, this will return `nil, err`.
// If the last value is a nil error it will be removed from the value list.
// Any remaining values will be converted and returned as `any` typed values.
func ParseReturn(ret []reflect.Value) ([]any, error) {
	if len(ret) == 0 {
		return nil, nil
	}
	last := ret[len(ret)-1]
	if last.Type().Implements(errorInterface) {
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}
		ret = ret[:len(ret)-1]
	}
	out := make([]any, len(ret))
	for i, r := range ret {
		out[i] = r.Interface()
	}
	return out, nil
}
