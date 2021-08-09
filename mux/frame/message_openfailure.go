package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type OpenFailureMessage struct {
	ChannelID uint32
}

func (msg OpenFailureMessage) String() string {
	return fmt.Sprintf("{OpenFailureMessage ChannelID:%d}", msg.ChannelID)
}

func (msg OpenFailureMessage) Channel() (uint32, bool) {
	return msg.ChannelID, true
}

func (msg OpenFailureMessage) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(msgChannelOpenFailure)
	binary.Write(buf, binary.BigEndian, msg)
	return buf.Bytes()
}
