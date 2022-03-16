package mux

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/progrium/qtalk-go/mux/frame"
)

const (
	minPacketLength = 9
	maxPacketLength = 1 << 31

	// channelMaxPacket contains the maximum number of bytes that will be
	// sent in a single packet. As per RFC 4253, section 6.1, 32k is also
	// the minimum.
	channelMaxPacket = 1 << 15
	// We follow OpenSSH here.
	channelWindowSize = 64 * channelMaxPacket

	// chanSize sets the amount of buffering qmux connections. This is
	// primarily for testing: setting chanSize=0 uncovers deadlocks more
	// quickly.
	chanSize = 16
)

// Session is a bi-directional channel muxing session on a given transport.
type Session struct {
	t     io.ReadWriteCloser
	chans chanList

	enc *frame.Encoder
	dec *frame.Decoder

	inbox chan *Channel

	errCond *sync.Cond
	err     error
	closeCh chan bool
}

// NewSession returns a session that runs over the given transport.
func New(t io.ReadWriteCloser) *Session {
	if t == nil {
		return nil
	}
	s := &Session{
		t:       t,
		enc:     frame.NewEncoder(t),
		dec:     frame.NewDecoder(t),
		inbox:   make(chan *Channel, chanSize),
		errCond: sync.NewCond(new(sync.Mutex)),
		closeCh: make(chan bool, 1),
	}
	go s.loop()
	return s
}

// Close closes the underlying transport.
func (s *Session) Close() error {
	s.t.Close()
	return nil
}

// Wait blocks until the transport has shut down, and returns the
// error causing the shutdown.
func (s *Session) Wait() error {
	s.errCond.L.Lock()
	defer s.errCond.L.Unlock()
	for s.err == nil {
		s.errCond.Wait()
	}
	return s.err
}

// Accept waits for and returns the next incoming channel.
func (s *Session) Accept() (*Channel, error) {
	select {
	case ch := <-s.inbox:
		return ch, nil
	case <-s.closeCh:
		return nil, io.EOF
	}
}

// Open establishes a new channel with the other end.
func (s *Session) Open(ctx context.Context) (*Channel, error) {
	ch := s.newChannel(channelOutbound)
	ch.maxIncomingPayload = channelMaxPacket

	if err := s.enc.Encode(frame.OpenMessage{
		WindowSize:    ch.myWindow,
		MaxPacketSize: ch.maxIncomingPayload,
		SenderID:      ch.localId,
	}); err != nil {
		return nil, err
	}

	var m frame.Message

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case m = <-ch.msg:
		if m == nil {
			// channel was closed before open got a response,
			// typically meaning the session/conn was closed.
			return nil, net.ErrClosed
		}
	}

	switch msg := m.(type) {
	case *frame.OpenConfirmMessage:
		return ch, nil
	case *frame.OpenFailureMessage:
		return nil, fmt.Errorf("qmux: channel open failed on remote side")
	default:
		return nil, fmt.Errorf("qmux: unexpected packet in response to channel open: %v", msg)
	}
}

func (s *Session) newChannel(direction channelDirection) *Channel {
	ch := &Channel{
		remoteWin: window{Cond: sync.NewCond(new(sync.Mutex))},
		myWindow:  channelWindowSize,
		pending:   newBuffer(),
		direction: direction,
		msg:       make(chan frame.Message, chanSize),
		session:   s,
		packetBuf: make([]byte, 0),
	}
	ch.localId = s.chans.add(ch)
	return ch
}

// loop runs the connection machine. It will process packets until an
// error is encountered. To synchronize on loop exit, use session.Wait.
func (s *Session) loop() {
	var err error
	for err == nil {
		err = s.onePacket()
	}

	for _, ch := range s.chans.dropAll() {
		ch.close()
	}

	s.t.Close()
	s.closeCh <- true

	s.errCond.L.Lock()
	s.err = err
	s.errCond.Broadcast()
	s.errCond.L.Unlock()
}

// onePacket reads and processes one packet.
func (s *Session) onePacket() error {
	var err error
	var msg frame.Message

	msg, err = s.dec.Decode()
	if err != nil {
		return err
	}

	id, isChan := msg.Channel()
	if !isChan {
		return s.handleOpen(msg.(*frame.OpenMessage))
	}

	ch := s.chans.getChan(id)
	if ch == nil {
		return fmt.Errorf("qmux: invalid channel %d", id)
	}

	return ch.handle(msg)
}

// handleChannelOpen schedules a channel to be Accept()ed.
func (s *Session) handleOpen(msg *frame.OpenMessage) error {
	if msg.MaxPacketSize < minPacketLength || msg.MaxPacketSize > maxPacketLength {
		return s.enc.Encode(frame.OpenFailureMessage{
			ChannelID: msg.SenderID,
		})
	}

	c := s.newChannel(channelInbound)
	c.remoteId = msg.SenderID
	c.maxRemotePayload = msg.MaxPacketSize
	c.remoteWin.add(msg.WindowSize)
	c.maxIncomingPayload = channelMaxPacket
	s.inbox <- c

	return s.enc.Encode(frame.OpenConfirmMessage{
		ChannelID:     c.remoteId,
		SenderID:      c.localId,
		WindowSize:    c.myWindow,
		MaxPacketSize: c.maxIncomingPayload,
	})
}
