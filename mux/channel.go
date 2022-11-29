package mux

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/progrium/qtalk-go/mux/frame"
)

type channelDirection uint8

const (
	channelInbound channelDirection = iota
	channelOutbound
)

func min(a uint32, b int) uint32 {
	if a < uint32(b) {
		return a
	}
	return uint32(b)
}

type Channel interface {
	io.ReadWriteCloser
	ID() uint32
	CloseWrite() error
}

// channel is an implementation of the Channel interface that works
// with the session class.
type channel struct {

	// R/O after creation
	localId, remoteId uint32

	// maxIncomingPayload and maxRemotePayload are the maximum
	// payload sizes of normal and extended data packets for
	// receiving and sending, respectively. The wire packet will
	// be 9 or 13 bytes larger (excluding encryption overhead).
	maxIncomingPayload uint32
	maxRemotePayload   uint32

	session *session

	// direction contains either channelOutbound, for channels created
	// locally, or channelInbound, for channels created by the peer.
	direction channelDirection

	// Pending internal channel messages.
	msg chan frame.Message

	sentEOF bool

	// thread-safe data
	remoteWin window
	pending   *buffer

	// windowMu protects myWindow, the flow-control window.
	windowMu sync.Mutex
	myWindow uint32

	// writeMu serializes calls to session.conn.Write() and
	// protects sentClose and packetPool. This mutex must be
	// different from windowMu, as writePacket can block if there
	// is a key exchange pending.
	writeMu   sync.Mutex
	sentClose bool

	// packet buffer for writing
	packetBuf []byte
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
	return ch.send(frame.EOFMessage{
		ChannelID: ch.remoteId})
}

// Close signals end of channel use. No data may be sent after this
// call.
func (ch *channel) Close() error {
	return ch.send(frame.CloseMessage{
		ChannelID: ch.remoteId})
}

// Write writes len(data) bytes to the channel.
func (ch *channel) Write(data []byte) (n int, err error) {
	if ch.sentEOF {
		return 0, io.EOF
	}

	for len(data) > 0 {
		space := min(ch.maxRemotePayload, len(data))
		if space, err = ch.remoteWin.reserve(space); err != nil {
			return n, err
		}

		toSend := data[:space]

		if err = ch.session.enc.Encode(frame.DataMessage{
			ChannelID: ch.remoteId,
			Length:    uint32(len(toSend)),
			Data:      toSend,
		}); err != nil {
			return n, err
		}

		n += len(toSend)
		data = data[len(toSend):]
	}

	return n, err
}

// Read reads up to len(data) bytes from the channel.
func (c *channel) Read(data []byte) (n int, err error) {
	n, err = c.pending.Read(data)

	if n > 0 {
		err = c.adjustWindow(uint32(n))
		// sendWindowAdjust can return io.EOF if the remote
		// peer has closed the connection, however we want to
		// defer forwarding io.EOF to the caller of Read until
		// the buffer has been drained.
		if n > 0 && err == io.EOF {
			err = nil
		}
	}
	return n, err
}

// sends writes a message frame. If the message is a channel close, it updates
// sentClose. This method takes the lock c.writeMu.
func (ch *channel) send(msg frame.Message) error {
	ch.writeMu.Lock()
	defer ch.writeMu.Unlock()

	if ch.sentClose {
		return io.EOF
	}

	if _, ok := msg.(frame.CloseMessage); ok {
		ch.sentClose = true
	}

	return ch.session.enc.Encode(msg)
}

func (c *channel) adjustWindow(n uint32) error {
	c.windowMu.Lock()
	// Since myWindow is managed on our side, and can never exceed
	// the initial window setting, we don't worry about overflow.
	c.myWindow += uint32(n)
	c.windowMu.Unlock()
	return c.send(frame.WindowAdjustMessage{
		ChannelID:       c.remoteId,
		AdditionalBytes: uint32(n),
	})
}

func (c *channel) close() {
	c.pending.eof()
	close(c.msg)
	c.writeMu.Lock()
	// This is not necessary for a normal channel teardown, but if
	// there was another error, it is.
	c.sentClose = true
	c.writeMu.Unlock()
	// Unblock writers.
	c.remoteWin.close()
}

// responseMessageReceived is called when a success or failure message is
// received on a channel to check that such a message is reasonable for the
// given channel.
func (ch *channel) responseMessageReceived() error {
	if ch.direction == channelInbound {
		return errors.New("qmux: channel response message received on inbound channel")
	}
	return nil
}

func (ch *channel) handle(msg frame.Message) error {
	switch m := msg.(type) {
	case *frame.DataMessage:
		return ch.handleData(m)

	case *frame.CloseMessage:
		ch.send(frame.CloseMessage{
			ChannelID: ch.remoteId,
		})
		ch.session.chans.remove(ch.localId)
		ch.close()
		return nil

	case *frame.EOFMessage:
		ch.pending.eof()
		return nil

	case *frame.WindowAdjustMessage:
		if !ch.remoteWin.add(m.AdditionalBytes) {
			return fmt.Errorf("qmux: invalid window update for %d bytes", m.AdditionalBytes)
		}
		return nil

	case *frame.OpenConfirmMessage:
		if err := ch.responseMessageReceived(); err != nil {
			return err
		}
		if m.MaxPacketSize < minPacketLength || m.MaxPacketSize > maxPacketLength {
			return fmt.Errorf("qmux: invalid MaxPacketSize %d from peer", m.MaxPacketSize)
		}
		ch.remoteId = m.SenderID
		ch.maxRemotePayload = m.MaxPacketSize
		ch.remoteWin.add(m.WindowSize)
		ch.msg <- m
		return nil

	case *frame.OpenFailureMessage:
		if err := ch.responseMessageReceived(); err != nil {
			return err
		}
		ch.session.chans.remove(m.ChannelID)
		ch.msg <- m
		return nil

	default:
		return fmt.Errorf("qmux: invalid channel message %v", msg)
	}
}

func (ch *channel) handleData(msg *frame.DataMessage) error {
	if msg.Length > ch.maxIncomingPayload {
		// TODO(hanwen): should send Disconnect?
		return errors.New("qmux: incoming packet exceeds maximum payload size")
	}

	if msg.Length != uint32(len(msg.Data)) {
		return errors.New("qmux: wrong packet length")
	}

	ch.windowMu.Lock()
	if ch.myWindow < msg.Length {
		ch.windowMu.Unlock()
		// TODO(hanwen): should send Disconnect with reason?
		return errors.New("qmux: remote side wrote too much")
	}
	ch.myWindow -= msg.Length
	ch.windowMu.Unlock()

	ch.pending.write(msg.Data)
	return nil
}
