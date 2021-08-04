package codec

import (
	"encoding/json"
	"io"
)

// JSONCodec provides a codec API for the standard library JSON encoder and decoder.
type JSONCodec struct{}

// Encoder returns a JSON encoder
func (c JSONCodec) Encoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

// Decoder returns a JSON decoder
func (c JSONCodec) Decoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
