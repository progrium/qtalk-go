package mux

const (
	channelOpen = iota + 100
	channelOpenConfirm
	channelOpenFailure
	channelData
	channelEOF
	channelClose
)

type Frame struct {
	_         struct{} `cbor:",toarray"`
	Type      byte
	ChannelID uint32
	SenderID  uint32
	Data      []byte `cbor:",omitempty"`
}
