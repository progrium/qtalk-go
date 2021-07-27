package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
)

// Decoder decodes messages given an io.Reader
type Decoder struct {
	r io.Reader
	sync.Mutex
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (dec *Decoder) Decode() (Message, error) {
	dec.Lock()
	defer dec.Unlock()

	var msgNum [1]byte
	_, err := io.ReadFull(dec.r, msgNum[:])
	if err != nil {
		var syscallErr *os.SyscallError
		if errors.As(err, &syscallErr) && syscallErr.Err == syscall.ECONNRESET {
			return nil, io.EOF
		}
		return nil, err
	}

	var msg Message
	msg, err = messageFrom(msgNum)
	if err != nil {
		return nil, err
	}

	if msgNum[0] == msgChannelData {
		var data struct {
			ChannelID uint32
			Length    uint32
		}
		if err := binary.Read(dec.r, binary.BigEndian, &data); err != nil {
			return nil, err
		}
		dataMsg := msg.(*DataMessage)
		dataMsg.ChannelID = data.ChannelID
		dataMsg.Length = data.Length
		dataMsg.Data = make([]byte, data.Length)
		_, err := io.ReadFull(dec.r, dataMsg.Data)
		if err != nil {
			return nil, err
		}
	} else {
		if err := binary.Read(dec.r, binary.BigEndian, msg); err != nil {
			return nil, err
		}
	}

	if Debug != nil {
		fmt.Fprintln(Debug, ">>DEC", msg)
	}

	return msg, nil
}

func messageFrom(num [1]byte) (Message, error) {
	switch num[0] {
	case msgChannelOpen:
		return new(OpenMessage), nil
	case msgChannelData:
		return new(DataMessage), nil
	case msgChannelOpenConfirm:
		return new(OpenConfirmMessage), nil
	case msgChannelOpenFailure:
		return new(OpenFailureMessage), nil
	case msgChannelWindowAdjust:
		return new(WindowAdjustMessage), nil
	case msgChannelEOF:
		return new(EOFMessage), nil
	case msgChannelClose:
		return new(CloseMessage), nil
	default:
		return nil, fmt.Errorf("qtalk: unexpected message type %d", num[0])
	}
}
