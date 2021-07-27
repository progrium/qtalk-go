package codec

const (
	msgChannelOpen = iota + 100
	msgChannelOpenConfirm
	msgChannelOpenFailure
	msgChannelWindowAdjust
	msgChannelData
	msgChannelEOF
	msgChannelClose
)

type Message interface {
	Channel() (uint32, bool)
	String() string
	Bytes() []byte
}
