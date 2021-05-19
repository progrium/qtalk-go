package codec

import (
	"io"
)

type Encoder interface {
	Encode(v interface{}) error
}

type Decoder interface {
	Decode(v interface{}) error
}

type Codec interface {
	Encoder(w io.Writer) Encoder
	Decoder(r io.Reader) Decoder
}
