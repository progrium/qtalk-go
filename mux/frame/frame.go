// Package frame implements encoding and decoding of qmux message frames.
package frame

import "io"

var (
	// Debug can be set to get message frames as they're encoded and decoded
	Debug io.Writer
)
