package fn

import (
	"fmt"
	"reflect"
	"testing"
)

func fatal(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func callParseReturn(fn interface{}, args []reflect.Value) ([]any, error) {
	ret := reflect.ValueOf(fn).Call(args)
	return ParseReturn(ret)
}

func values(args ...any) []reflect.Value {
	r := make([]reflect.Value, len(args))
	for i, a := range args {
		r[i] = reflect.ValueOf(a)
	}
	return r
}

func equal(expected, actual []any) bool {
	if len(expected) == 0 {
		return len(actual) == 0
	}
	return reflect.DeepEqual(expected, actual)
}

func TestParseReturn(t *testing.T) {
	tests := []struct {
		name        string
		fn          interface{}
		args        []reflect.Value
		expected    []any
		expectedErr bool
	}{
		{"no return values", func() {}, nil, nil, false},
		{
			"single value return", func(i int) int { return i * 2 }, values(int(21)),
			[]any{int(42)}, false,
		},
		{
			"multiple value return", func(i int) (int, float64) {
				return i * 2, float64(i) / 2
			}, values(int(21)),
			[]any{int(42), float64(10.5)}, false,
		},

		{"return nil error", func() error { return nil }, nil, nil, false},
		{"return non-nil error", func() error { return fmt.Errorf("an error") }, nil, nil, true},

		{
			"single value with nil error", func() (int, error) { return 42, nil }, nil,
			[]any{int(42)}, false,
		},
		{
			"single value with non-nil error", func() (int, error) { return 42, fmt.Errorf("an error") }, nil,
			nil, true,
		},

		{
			"multiple value with nil error", func() (int, float64, error) { return 42, 0.5, nil }, nil,
			[]any{int(42), float64(0.5)}, false,
		},
		{
			"multiple value with non-nil error", func() (int, float64, error) { return 42, 0.5, fmt.Errorf("an error") }, nil,
			nil, true,
		},

		{
			"return error as value", func() any { return fmt.Errorf("an error") }, nil,
			[]any{fmt.Errorf("an error")}, false,
		},
	}
	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			actual, err := callParseReturn(td.fn, td.args)
			if !td.expectedErr {
				fatal(err, t)
			} else if err == nil {
				t.Fatalf("expected an error")
			}
			if !equal(td.expected, actual) {
				t.Errorf("expected: %v\ngot: %v", td.expected, actual)
			}
		})
	}
}
