// Package codec implements encoding and decoding of qmux messages.
package codec

import "io"

var (
	// Debug can be set to get messages as they're encoded and decoded
	Debug io.Writer
)
