package exp

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/progrium/qtalk-go/fn"
	"github.com/progrium/qtalk-go/rpc"
	"github.com/rs/xid"
)

// Ptr represents a remote function pointer.
type Ptr struct {
	Ptr    string     `json:"$fnptr" mapstructure:"$fnptr"`
	Caller rpc.Caller `json:"-"`
	fn     interface{}
}

// Call uses the Ptr Caller to call this remote function using the Ptr ID as the selector.
// If Caller is not set on Ptr, Call will panic. Use SetCallers on incoming parameters that
// may include Ptrs.
func (p *Ptr) Call(ctx context.Context, params, reply interface{}) (*rpc.Response, error) {
	return p.Caller.Call(ctx, p.Ptr, params, reply)
}

// Callback wraps a function in a Ptr giving it a 20 character unique string ID.
// A unique ID is created with every call, so should only be called once for a
// given function.
func Callback(fn interface{}) *Ptr {
	return &Ptr{
		Ptr: xid.New().String(),
		fn:  fn,
	}
}

// SetCallers will set the Caller for any Ptrs found in the value using PtrsFrom, as well as any
// Ptrs encoded as maps, identifying them with the special key "$fnptr". Without Callers, Ptrs
// will panic when using Call. This is often used on map[string]interface{} parameters before
// running through something like github.com/mitchellh/mapstructure.
func SetCallers(v interface{}, c rpc.Caller) []string {
	var ptrs []string
	for _, ptr := range PtrsFrom(v) {
		ptr.Caller = c
		ptrs = append(ptrs, ptr.Ptr)
	}
	walk(reflect.ValueOf(v), []string{}, func(v reflect.Value, parent reflect.Value, path []string) error {
		if path[len(path)-1] == "$fnptr" {
			parent.SetMapIndex(reflect.ValueOf("Caller"), reflect.ValueOf(c))
			ptrs = append(ptrs, v.String())
		}
		return nil
	})
	return ptrs
}

// RegisterPtrs will register handlers on the RespondMux for Ptrs found using PtrsFrom on the value.
// This is often called before making an RPC call that will include Ptr callbacks. It can safely be
// called more than once for the same Ptrs, as it will only register handlers if they have not been
// registered. They are registered on the RespondMux using the Ptr ID and a handler from HandlerFrom
// on the Ptr function.
func RegisterPtrs(m *rpc.RespondMux, v interface{}) {
	ptrs := PtrsFrom(v)
	for _, ptr := range ptrs {
		if h, _ := m.Match(ptr.Ptr); h == nil {
			m.Handle(ptr.Ptr, fn.HandlerFrom(ptr.fn))
		}
	}
}

// UnregisterPtrs will remove handlers from the RespondMux matching Ptr IDs found using PtrsFrom on the value.
func UnregisterPtrs(m *rpc.RespondMux, v interface{}) {
	ptrs := PtrsFrom(v)
	for _, ptr := range ptrs {
		if h, _ := m.Match(ptr.Ptr); h != nil {
			m.Remove(ptr.Ptr)
		}
	}
}

// PtrsFrom collects Ptrs from walking exported struct fields, slice/array elements, map values, and pointers in a value.
func PtrsFrom(v interface{}) (ptrs []*Ptr) {
	typ := reflect.TypeOf(&Ptr{})
	walk(reflect.ValueOf(v), []string{}, func(v reflect.Value, parent reflect.Value, path []string) error {
		if v.Type() == typ {
			vv := v.Interface().(*Ptr)
			if v.IsNil() {
				return nil
			}
			ptrs = append(ptrs, vv)
		}
		return nil
	})
	return
}

func walk(v reflect.Value, path []string, visitor func(v reflect.Value, parent reflect.Value, path []string) error) error {
	for _, k := range keys(v) {
		subpath := append(path, k)
		vv := prop(v, k)
		if !vv.IsValid() {
			continue
		}
		if err := visitor(vv, v, subpath); err != nil {
			return err
		}
		if err := walk(vv, subpath, visitor); err != nil {
			return err
		}
	}
	return nil
}

func prop(robj reflect.Value, key string) reflect.Value {
	rtyp := robj.Type()
	switch rtyp.Kind() {
	case reflect.Slice, reflect.Array:
		idx, err := strconv.Atoi(key)
		if err != nil {
			panic("non-numeric index given for slice")
		}
		rval := robj.Index(idx)
		if rval.IsValid() {
			return reflect.ValueOf(rval.Interface())
		}
	case reflect.Ptr:
		return prop(robj.Elem(), key)
	case reflect.Map:
		rval := robj.MapIndex(reflect.ValueOf(key))
		if rval.IsValid() {
			return reflect.ValueOf(rval.Interface())
		}
	case reflect.Struct:
		rval := robj.FieldByName(key)
		if rval.IsValid() {
			return rval
		}
		for i := 0; i < rtyp.NumField(); i++ {
			field := rtyp.Field(i)
			tag := strings.Split(field.Tag.Get("json"), ",")
			if tag[0] == key || field.Name == key {
				return robj.FieldByName(field.Name)
			}
		}
		panic("struct field not found: " + key)
	}
	//spew.Dump(robj, key)
	panic("unexpected kind: " + rtyp.Kind().String())
}

func keys(v reflect.Value) []string {
	switch v.Type().Kind() {
	case reflect.Map:
		var keys []string
		for _, key := range v.MapKeys() {
			k, ok := key.Interface().(string)
			if !ok {
				continue
			}
			keys = append(keys, k)
		}
		sort.Sort(sort.StringSlice(keys))
		return keys
	case reflect.Struct:
		t := v.Type()
		var f []string
		for i := 0; i < t.NumField(); i++ {
			name := t.Field(i).Name
			// first letter capitalized means exported
			if name[0] == strings.ToUpper(name)[0] {
				f = append(f, name)
			}
		}
		return f
	case reflect.Slice, reflect.Array:
		var k []string
		for n := 0; n < v.Len(); n++ {
			k = append(k, strconv.Itoa(n))
		}
		return k
	case reflect.Ptr:
		if !v.IsNil() {
			return keys(v.Elem())
		}
		return []string{}
	case reflect.String, reflect.Bool, reflect.Float64, reflect.Float32, reflect.Interface:
		return []string{}
	default:
		fmt.Fprintf(os.Stderr, "unexpected type: %s\n", v.Type().Kind())
		return []string{}
	}
}
