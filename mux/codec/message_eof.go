package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type EOFMessage struct {
	ChannelID uint32
}

func (msg EOFMessage) String() string {
	return fmt.Sprintf("{EOFMessage ChannelID:%d}", msg.ChannelID)
}

func (msg EOFMessage) Channel() (uint32, bool) {
	return msg.ChannelID, true
}

func (msg EOFMessage) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(msgChannelEOF)
	binary.Write(buf, binary.BigEndian, msg)
	return buf.Bytes()
}
