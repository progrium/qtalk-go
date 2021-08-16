package peer

import (
	"fmt"

	"github.com/progrium/qtalk-go/codec"
	"github.com/progrium/qtalk-go/mux"
	"github.com/progrium/qtalk-go/transport"
)

// A Dialer connects to address and establishes a mux.Session
type Dialer func(addr string) (*mux.Session, error)

// Dialers is map of transport strings to Dialers
// and includes all builtin transports
var Dialers map[string]Dialer

func init() {
	Dialers = map[string]Dialer{
		"tcp":  transport.DialTCP,
		"unix": transport.DialUnix,
		"ws":   transport.DialWS,
		"stdio": func(_ string) (*mux.Session, error) {
			return transport.DialStdio()
		},
	}
}

// Dial connects to a remote address using a registered transport and returns a Peer.
// Available transports are "tcp", "unix", "ws", and "stdio". In the case of "stdio",
// the addr can be left an empty string.
func Dial(transport, addr string, codec codec.Codec) (*Peer, error) {
	d, ok := Dialers[transport]
	if !ok {
		return nil, fmt.Errorf("transport '%s' not in available in Dialers", transport)
	}
	sess, err := d(addr)
	if err != nil {
		return nil, err
	}
	return New(sess, codec), nil
}
