package codec

import (
	"bytes"
	"testing"
)

type testData struct {
	Map map[string]bool
	Arr []int
}

func TestJSONCodec(t *testing.T) {
	c := &JSONCodec{}
	var buf bytes.Buffer

	if err := c.Encoder(&buf).Encode(testData{
		Map: map[string]bool{"true": true, "false": false},
		Arr: []int{1, 2, 3},
	}); err != nil {
		t.Fatal(err)
	}

	var data testData
	if err := c.Decoder(&buf).Decode(&data); err != nil {
		t.Fatal(err)
	}

	if data.Map["true"] != true || data.Arr[2] != 3 {
		t.Fatal("unexpected data:", data)
	}
}
