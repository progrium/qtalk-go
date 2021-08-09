package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type OpenMessage struct {
	SenderID      uint32
	WindowSize    uint32
	MaxPacketSize uint32
}

func (msg OpenMessage) String() string {
	return fmt.Sprintf("{OpenMessage SenderID:%d WindowSize:%d MaxPacketSize:%d}",
		msg.SenderID, msg.WindowSize, msg.MaxPacketSize)
}

func (msg OpenMessage) Channel() (uint32, bool) {
	return 0, false
}

func (msg OpenMessage) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(msgChannelOpen)
	binary.Write(buf, binary.BigEndian, msg)
	return buf.Bytes()
}
