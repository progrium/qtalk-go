package frame

import (
	"encoding/binary"
	"fmt"
)

type DataMessage struct {
	ChannelID uint32
	Length    uint32
	Data      []byte
}

func (msg DataMessage) String() string {
	return fmt.Sprintf("{DataMessage ChannelID:%d Length:%d Data: ... }",
		msg.ChannelID, msg.Length)
}

func (msg DataMessage) Channel() (uint32, bool) {
	return msg.ChannelID, true
}

func (msg DataMessage) Bytes() []byte {
	packet := make([]byte, 9)
	packet[0] = msgChannelData
	binary.BigEndian.PutUint32(packet[1:5], msg.ChannelID)
	binary.BigEndian.PutUint32(packet[5:9], msg.Length)
	return append(packet, msg.Data...)
}
