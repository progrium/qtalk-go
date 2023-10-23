package webrtc

import (
	"context"
	"fmt"
	"io"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"github.com/progrium/qtalk-go/mux"
)

var rtcapi = func() *webrtc.API {
	s := webrtc.SettingEngine{}
	s.DetachDataChannels()
	return webrtc.NewAPI(webrtc.WithSettingEngine(s))
}()

func New(peer *webrtc.PeerConnection) mux.Session {
	// we probably want client to make an offer, then we provide an answer on a
	// new peer connection
	// maybe maintain a pool of peer connections to have ready to connect?

	// wait until we get a connection before returning?
	channels := make(chan *webrtc.DataChannel)
	peer.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			// should we use the meta channel for control signals, or close it?
			if d.Label() == "qtalk" {
				channels <- d
			}
		})
	})
	return &session{peer, channels, make(chan struct{})}
}

// let's start with simpler version where we make one offer and wait for an
// answer

// need to establish a single channel for the answer and exchanging ICE
// candidates

// but we can wait for ICE to complete, then print the local description

func Offer(cfg webrtc.Configuration) (*webrtc.PeerConnection, error) {
	peer, err := rtcapi.NewPeerConnection(cfg)
	if err != nil {
		return nil, err
	}
	// WebRTC requires having at least one channel before creating the offer.
	// For now this is unused, but we could use this as an internal control
	// channel. Maybe we can close it after the connection is established?
	_, err = peer.CreateDataChannel("qtalk-meta", nil)
	if err != nil {
		peer.Close()
		return nil, err
	}
	offer, err := peer.CreateOffer(nil)
	if err != nil {
		peer.Close()
		return nil, err
	}
	if err := peer.SetLocalDescription(offer); err != nil {
		peer.Close()
		return nil, err
	}
	return peer, nil
}

func Answer(cfg webrtc.Configuration, offer webrtc.SessionDescription) (*webrtc.PeerConnection, error) {
	peer, err := rtcapi.NewPeerConnection(cfg)
	if err != nil {
		return nil, err
	}
	if err := peer.SetRemoteDescription(offer); err != nil {
		peer.Close()
		return nil, err
	}
	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		peer.Close()
		return nil, err
	}
	if err := peer.SetLocalDescription(answer); err != nil {
		peer.Close()
		return nil, err
	}
	return peer, nil
}

type session struct {
	peer     *webrtc.PeerConnection
	channels chan *webrtc.DataChannel
	done     chan struct{}
}

func (s *session) Close() error {
	return s.peer.Close()
}

func (s *session) Accept() (mux.Channel, error) {
	// TODO select on done channel
	select {
	case ch := <-s.channels:
		return newChannel(ch)
	case <-s.done:
		return nil, fmt.Errorf("session closed")
	}
}

func (s *session) Open(ctx context.Context) (mux.Channel, error) {
	d, err := s.peer.CreateDataChannel("qtalk", nil)
	if err != nil {
		return nil, err
	}
	opened := make(chan struct{})
	d.OnOpen(func() { close(opened) })
	select {
	case <-opened:
		return newChannel(d)
	case <-ctx.Done():
		d.Close()
		return nil, ctx.Err()
	}
}

func (s *session) Wait() error {
	panic("not implemented")
}

func newChannel(ch *webrtc.DataChannel) (mux.Channel, error) {
	rwc, err := ch.Detach()
	if err != nil {
		return nil, err
	}
	return &channel{
		// id is assigned before calling OnOpen, so we expect it to be non-nil
		id:  uint32(*ch.ID()),
		ch:  ch,
		rwc: rwc,
	}, nil
}

type channel struct {
	id          uint32
	ch          *webrtc.DataChannel
	rwc         datachannel.ReadWriteCloser
	gotEOF      bool
	closedWrite bool
}

func (c *channel) ID() uint32 {
	return c.id
}

func (c *channel) Read(p []byte) (int, error) {
	if c.gotEOF {
		return 0, io.EOF
	}
	n, isString, err := c.rwc.ReadDataChannel(p)
	if err != nil {
		return n, err
	}
	if isString && string(p[:n]) == "EOF" {
		return 0, io.EOF
	}
	return n, nil
}

func (c *channel) Write(p []byte) (int, error) {
	if c.closedWrite {
		return 0, io.ErrClosedPipe
	}
	return c.rwc.Write(p)
}

func (c *channel) Close() error {
	return c.rwc.Close()
}

func (c *channel) CloseWrite() error {
	_, err := c.rwc.WriteDataChannel([]byte("EOF"), true)
	if err != nil {
		return err
	}
	c.closedWrite = true
	return nil
}

type Signaler interface {
}
