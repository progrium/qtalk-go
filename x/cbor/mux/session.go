package mux

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/progrium/qtalk-go/mux"
)

var (
	// timeout for queuing a new channel to be `Accept`ed
	// use a `var` so that this can be overridden in tests
	acceptTimeout = 30 * time.Second
)

type session struct {
	t io.ReadWriteCloser

	chanMu      sync.Mutex
	chans       map[uint32]*channel
	chanCounter uint32

	enc *cbor.Encoder
	dec *cbor.Decoder

	inbox chan mux.Channel

	errCond *sync.Cond
	err     error
	closeCh chan bool
}

// NewSession returns a session that runs over the given transport.
func New(t io.ReadWriteCloser) mux.Session {
	if t == nil {
		return nil
	}
	s := &session{
		t:       t,
		enc:     cbor.NewEncoder(t),
		dec:     cbor.NewDecoder(t),
		inbox:   make(chan mux.Channel),
		chans:   make(map[uint32]*channel),
		errCond: sync.NewCond(new(sync.Mutex)),
		closeCh: make(chan bool, 1),
	}
	go s.loop()
	return s
}

// Close closes the underlying transport.
func (s *session) Close() error {
	s.t.Close()
	return nil
}

// Wait blocks until the transport has shut down, and returns the
// error causing the shutdown.
func (s *session) Wait() error {
	s.errCond.L.Lock()
	defer s.errCond.L.Unlock()
	for s.err == nil {
		s.errCond.Wait()
	}
	return s.err
}

// Accept waits for and returns the next incoming channel.
func (s *session) Accept() (mux.Channel, error) {
	select {
	case ch := <-s.inbox:
		return ch, nil
	case <-s.closeCh:
		return nil, io.EOF
	}
}

// Open establishes a new channel with the other end.
func (s *session) Open(ctx context.Context) (mux.Channel, error) {
	ch := s.newChannel()
	if err := s.enc.Encode(Frame{
		Type:     channelOpen,
		SenderID: ch.localId,
	}); err != nil {
		return nil, err
	}

	var f Frame
	var ok bool

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case f, ok = <-ch.frames:
		if !ok {
			// channel was closed before open got a response,
			// typically meaning the session/conn was closed.
			return nil, net.ErrClosed
		}
	}

	switch f.Type {
	case channelOpenConfirm:
		return ch, nil
	case channelOpenFailure:
		return nil, fmt.Errorf("cmux: channel open failed on remote side")
	default:
		return nil, fmt.Errorf("cmux: unexpected packet in response to channel open: %v", f)
	}
}

func (s *session) newChannel() *channel {
	ch := &channel{
		pending: newBuffer(),
		frames:  make(chan Frame, 0),
		session: s,
	}
	s.chanMu.Lock()
	s.chanCounter++
	ch.localId = s.chanCounter
	s.chans[ch.localId] = ch
	s.chanMu.Unlock()
	return ch
}

// loop runs the connection machine. It will process packets until an
// error is encountered. To synchronize on loop exit, use session.Wait.
func (s *session) loop() {
	var err error
	for err == nil {
		err = s.onePacket()
	}
	//log.Println(err)

	s.chanMu.Lock()
	for _, ch := range s.chans {
		ch.close()
	}
	s.chans = make(map[uint32]*channel)
	s.chanCounter = 0
	s.chanMu.Unlock()

	s.t.Close()
	s.closeCh <- true

	s.errCond.L.Lock()
	s.err = err
	s.errCond.Broadcast()
	s.errCond.L.Unlock()
}

// onePacket reads and processes one packet.
func (s *session) onePacket() (err error) {
	var f Frame
	err = s.dec.Decode(&f)
	if err != nil {
		return err
	}

	if f.Type == channelOpen {
		c := s.newChannel()
		c.remoteId = f.SenderID
		t := time.NewTimer(acceptTimeout)
		defer t.Stop()
		select {
		case s.inbox <- c:
			return s.enc.Encode(Frame{
				Type:      channelOpenConfirm,
				ChannelID: c.remoteId,
				SenderID:  c.localId,
			})
		case <-t.C:
			return s.enc.Encode(Frame{
				Type:      channelOpenFailure,
				ChannelID: f.SenderID,
			})
		}
	}

	ch, ok := s.chans[f.ChannelID]
	if !ok {
		return fmt.Errorf("cmux: unknown channel %d", f.ChannelID)
	}
	return ch.handle(f)
}
