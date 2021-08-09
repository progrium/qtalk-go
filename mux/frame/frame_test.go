package frame

import (
	"bytes"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		in Message
		id uint32
		ok bool
	}{
		{
			in: CloseMessage{
				ChannelID: 10,
			},
			id: 10,
			ok: true,
		},
		{
			in: DataMessage{
				ChannelID: 10,
				Length:    5,
				Data:      []byte("Hello"),
			},
			id: 10,
			ok: true,
		},
		{
			in: EOFMessage{
				ChannelID: 10,
			},
			id: 10,
			ok: true,
		},
		{
			in: OpenMessage{
				SenderID:      10,
				WindowSize:    1024,
				MaxPacketSize: 1 << 31,
			},
			id: 0,
			ok: false,
		},
		{
			in: OpenConfirmMessage{
				ChannelID:     20,
				SenderID:      10,
				WindowSize:    1024,
				MaxPacketSize: 1 << 31,
			},
			id: 20,
			ok: true,
		},
		{
			in: OpenFailureMessage{
				ChannelID: 20,
			},
			id: 20,
			ok: true,
		},
		{
			in: WindowAdjustMessage{
				ChannelID:       20,
				AdditionalBytes: 1024,
			},
			id: 20,
			ok: true,
		},
	}
	for _, test := range tests {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		if err := enc.Encode(test.in); err != nil {
			t.Fatal(err)
		}
		dec := NewDecoder(&buf)
		m, err := dec.Decode()
		if err != nil {
			t.Fatal(err)
		}
		id, ok := m.Channel()
		if id != test.id {
			t.Fatal("id not equal")
		}
		if ok != test.ok {
			t.Fatal("ok not equal")
		}
		if m.String() == "" {
			t.Fatal("empty string representation")
		}
	}

}
