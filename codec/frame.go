package codec

import (
	"bytes"
	"encoding/binary"
	"io"
)

// length prefixed frame wrapper codec
type FrameCodec struct {
	Codec
}

func (c *FrameCodec) Encoder(w io.Writer) Encoder {
	return &frameEncoder{
		w: w,
		c: c.Codec,
	}
}

type frameEncoder struct {
	w io.Writer
	c Codec
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

func (c *FrameCodec) Decoder(r io.Reader) Decoder {
	return &frameDecoder{
		r: r,
		c: c.Codec,
	}
}

type frameDecoder struct {
	r io.Reader
	c Codec
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
