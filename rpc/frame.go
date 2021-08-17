package rpc

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/progrium/qtalk-go/codec"
)

// FrameCodec is a special codec used to actually read/write other
// codecs to a transport using a length prefix.
type FrameCodec struct {
	codec.Codec
}

// Encoder returns a frame encoder that first encodes a value
// to a buffer using the embedded codec, prepends the encoded value
// byte length as a four byte big endian uint32, then writes to
// the given Writer.
func (c *FrameCodec) Encoder(w io.Writer) codec.Encoder {
	return &frameEncoder{
		w: w,
		c: c.Codec,
	}
}

type frameEncoder struct {
	w io.Writer
	c codec.Codec
}

func (e *frameEncoder) Encode(v interface{}) error {
	var buf bytes.Buffer
	enc := e.c.Encoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return err
	}
	b := buf.Bytes()
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, uint32(len(b)))
	_, err = e.w.Write(append(prefix, b...))
	if err != nil {
		return err
	}
	return nil
}

// Decoder returns a frame decoder that first reads a four byte frame
// length value used to read the rest of the frame, then uses the
// embedded codec to decode those bytes into a value.
func (c *FrameCodec) Decoder(r io.Reader) codec.Decoder {
	return &frameDecoder{
		r: r,
		c: c.Codec,
	}
}

type frameDecoder struct {
	r io.Reader
	c codec.Codec
}

func (d *frameDecoder) Decode(v interface{}) error {
	prefix := make([]byte, 4)
	_, err := io.ReadFull(d.r, prefix)
	if err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(prefix)
	buf := make([]byte, size)
	_, err = io.ReadFull(d.r, buf)
	if err != nil {
		return err
	}
	dec := d.c.Decoder(bytes.NewBuffer(buf))
	err = dec.Decode(v)
	if err != nil {
		return err
	}
	return nil
}
