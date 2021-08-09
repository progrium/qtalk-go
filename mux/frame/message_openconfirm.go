package frame

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type OpenConfirmMessage struct {
	ChannelID     uint32
	SenderID      uint32
	WindowSize    uint32
	MaxPacketSize uint32
}

func (msg OpenConfirmMessage) String() string {
	return fmt.Sprintf("{OpenConfirmMessage ChannelID:%d SenderID:%d WindowSize:%d MaxPacketSize:%d}",
		msg.ChannelID, msg.SenderID, msg.WindowSize, msg.MaxPacketSize)
}

func (msg OpenConfirmMessage) Channel() (uint32, bool) {
	return msg.ChannelID, true
}

func (msg OpenConfirmMessage) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(msgChannelOpenConfirm)
	binary.Write(buf, binary.BigEndian, msg)
	return buf.Bytes()
}
