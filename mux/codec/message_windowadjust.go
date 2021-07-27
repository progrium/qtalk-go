package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type WindowAdjustMessage struct {
	ChannelID       uint32
	AdditionalBytes uint32
}

func (msg WindowAdjustMessage) String() string {
	return fmt.Sprintf("{WindowAdjustMessage ChannelID:%d AdditionalBytes:%d}",
		msg.ChannelID, msg.AdditionalBytes)
}

func (msg WindowAdjustMessage) Channel() (uint32, bool) {
	return msg.ChannelID, true
}

func (msg WindowAdjustMessage) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(msgChannelWindowAdjust)
	binary.Write(buf, binary.BigEndian, msg)
	return buf.Bytes()
}
