package frame

import (
	"fmt"
	"io"
	"sync"
)

// Encoder encodes messages given an io.Writer
type Encoder struct {
	w io.Writer
	sync.Mutex
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (enc *Encoder) Encode(msg Message) error {
	enc.Lock()
	defer enc.Unlock()

	if Debug != nil {
		fmt.Fprintln(Debug, "<<ENC", msg)
	}

	_, err := enc.w.Write(msg.Bytes())
	return err
}
