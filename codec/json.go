package codec

import (
	"encoding/json"
	"io"
)

type JSONCodec struct{}

func (c JSONCodec) Encoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

func (c JSONCodec) Decoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
