package mux

import (
	"fmt"
	"io"
	"sync"
)

// channel is an implementation of the Channel interface that works
// with the session class.
type channel struct {
	localId, remoteId uint32
	session           *session

	// Pending internal channel frames.
	frames chan Frame

	sentEOF bool

	// thread-safe data
	pending *buffer

	// writeMu serializes calls to session.conn.Write() and
	// protects sentClose.
	writeMu   sync.Mutex
	sentClose bool
}

// ID returns the unique identifier of this channel
// within the session
func (ch *channel) ID() uint32 {
	return ch.localId
}

// CloseWrite signals the end of sending data.
// The other side may still send data
func (ch *channel) CloseWrite() error {
	ch.sentEOF = true
	return ch.send(Frame{
		Type:      channelEOF,
		ChannelID: ch.remoteId,
	})
}

// Close signals end of channel use. No data may be sent after this
// call.
func (ch *channel) Close() error {
	return ch.send(Frame{
		Type:      channelClose,
		ChannelID: ch.remoteId,
	})
}

// Write writes len(data) bytes to the channel.
func (ch *channel) Write(data []byte) (n int, err error) {
	if ch.sentEOF {
		return 0, io.EOF
	}

	err = ch.session.enc.Encode(Frame{
		Type:      channelData,
		ChannelID: ch.remoteId,
		Data:      data,
	})

	return n, err
}

// Read reads up to len(data) bytes from the channel.
func (c *channel) Read(data []byte) (n int, err error) {
	return c.pending.Read(data)
}

// sends writes a message frame. If the message is a channel close, it updates
// sentClose. This method takes the lock c.writeMu.
func (ch *channel) send(f Frame) error {
	ch.writeMu.Lock()
	defer ch.writeMu.Unlock()

	if ch.sentClose {
		return io.EOF
	}

	if f.Type == channelClose {
		ch.sentClose = true
	}

	return ch.session.enc.Encode(f)
}

func (c *channel) close() {
	c.pending.eof()
	close(c.frames)
	c.writeMu.Lock()
	// This is not necessary for a normal channel teardown, but if
	// there was another error, it is.
	c.sentClose = true
	c.writeMu.Unlock()
}

func (ch *channel) handle(f Frame) error {
	switch f.Type {
	case channelData:
		ch.pending.write(f.Data)
		return nil

	case channelClose:
		ch.send(Frame{
			Type:      channelClose,
			ChannelID: ch.remoteId,
		})
		ch.session.chanMu.Lock()
		delete(ch.session.chans, ch.localId)
		ch.session.chanMu.Unlock()
		ch.close()
		return nil

	case channelEOF:
		ch.pending.eof()
		return nil

	case channelOpenConfirm:
		ch.remoteId = f.SenderID
		ch.frames <- f
		return nil

	case channelOpenFailure:
		ch.session.chanMu.Lock()
		delete(ch.session.chans, f.ChannelID)
		ch.session.chanMu.Unlock()
		ch.frames <- f
		return nil

	default:
		return fmt.Errorf("cmux: invalid channel frame %v", f)
	}
}
