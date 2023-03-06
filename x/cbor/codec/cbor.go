package codec

import (
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/progrium/qtalk-go/codec"
)

// CBORCodec provides a codec API for a CBOR encoder and decoder.
type CBORCodec struct{}

// Encoder returns a CBOR encoder
func (c CBORCodec) Encoder(w io.Writer) codec.Encoder {
	return cbor.NewEncoder(w)
}

// Decoder returns a CBOR decoder
func (c CBORCodec) Decoder(r io.Reader) codec.Decoder {
	return cbor.NewDecoder(r)
}
