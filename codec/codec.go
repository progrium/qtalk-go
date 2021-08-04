package codec

import (
	"io"
)

type Encoder interface {
	// Encode writes an encoding of v to its Writer.
	Encode(v interface{}) error
}

type Decoder interface {
	// Decode reads the next encoded value from its Reader and stores it in the value pointed to by v.
	Decode(v interface{}) error
}

// Codec returns an Encoder or Decoder given a Writer or Reader.
type Codec interface {
	Encoder(w io.Writer) Encoder
	Decoder(r io.Reader) Decoder
}
