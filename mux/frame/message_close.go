package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type CloseMessage struct {
	ChannelID uint32
}

func (msg CloseMessage) String() string {
	return fmt.Sprintf("{CloseMessage ChannelID:%d}", msg.ChannelID)
}

func (msg CloseMessage) Channel() (uint32, bool) {
	return msg.ChannelID, true
}

func (msg CloseMessage) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(msgChannelClose)
	binary.Write(buf, binary.BigEndian, msg)
	return buf.Bytes()
}
