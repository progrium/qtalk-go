package libp2p

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/progrium/qtalk-go/mux"
)

type channel struct {
	stream network.Stream
}

var _ mux.Channel = (*channel)(nil)

func (c *channel) ID() uint32 {
	// there's c.stream.ID() but that's a string, probably need a counter instead
	panic("not implemented")
}

func (c *channel) Read(p []byte) (n int, err error) {
	// If we get EOF, notify session that we're closed
	return c.stream.Read(p)
}

func (c *channel) Write(p []byte) (n int, err error) {
	// If we get EOF, notify session that we're closed
	return c.stream.Write(p)
}

func (c *channel) Close() error {
	// TODO notify session that we're closed?
	return c.stream.Close()
}

func (c *channel) CloseWrite() error {
	return c.stream.CloseWrite()
}

type session struct {
	host  myHost
	peer  peer.ID
	inbox chan network.Stream
}

var _ mux.Session = (*session)(nil)

func (s *session) Close() error {
	// FIXME tell channels to close
	return nil
}

func (s *session) Accept() (mux.Channel, error) {
	stream := <-s.inbox
	return &channel{stream}, nil
}

func (s *session) Open(ctx context.Context) (mux.Channel, error) {
	stream, err := s.host.NewStream(ctx, s.peer, protocolID)
	if err != nil {
		return nil, err
	}
	return &channel{stream}, nil
}

func (s *session) Wait() error {
	panic("not implemented")
}

type myHost interface {
	// ID returns the (local) peer.ID associated with this Host
	ID() peer.ID

	// NewStream opens a new stream to given peer p, and writes a p2p/protocol
	// header with given ProtocolID. If there is no connection to p, attempts
	// to create one. If ProtocolID is "", writes no header.
	// (Threadsafe)
	NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error)

	// SetStreamHandler sets the protocol handler on the Host's Mux.
	// This is equivalent to:
	//   host.Mux().SetHandler(proto, handler)
	// (Threadsafe)
	SetStreamHandler(pid protocol.ID, handler network.StreamHandler)

	// Close shuts down the host, its Network, and services.
	Close() error
}

type conn2 struct {
	mu       sync.Mutex
	sessions map[peer.ID]*session
	host     myHost
	inbox    chan network.Stream
	disc     discoverer
	cancel   context.CancelFunc
}

// Handshake on initial stream when opening a session by closing the write end
// then waiting for the other end to close its write end. Shuts down the
// opening stream while acknowledging that the other side has received the
// stream.
func closeACK(stream network.Stream) error {
	if err := stream.CloseWrite(); err != nil {
		return err
	}
	var buf [1]byte
	_, err := stream.Read(buf[:])
	if err == nil {
		return fmt.Errorf("expected EOF, but got data")
	}
	if err != io.EOF {
		return err
	}
	return stream.Close()
}

func Dial2(rendezvous string) (mux.Session, error) {
	dialTimeout := time.Second * 10
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	return Dial2Context(ctx, rendezvous)
}

func Dial2Context(ctx context.Context, rendezvous string) (mux.Session, error) {
	host, stream, err := connectToPeer(ctx, rendezvous)
	if err != nil {
		return nil, err
	}
	session := &session{
		host:  host,
		peer:  stream.Conn().RemotePeer(),
		inbox: make(chan network.Stream),
	}
	if err := closeACK(stream); err != nil {
		return nil, err
	}

	host.SetStreamHandler(protocolID, func(stream network.Stream) {
		if stream.Conn().RemotePeer() != session.peer {
			stream.Reset()
			return
		}
		session.inbox <- stream
	})
	return session, nil
}

func Listen2(ctx context.Context, rendezvous string) (*conn2, error) {
	host, disc, err := p2p(ctx)
	if err != nil {
		return nil, err
	}
	ctx2, cancel := context.WithCancel(ctx)
	dutil.Advertise(ctx2, disc, rendezvous)
	logger.Info("Listening as:", host.ID())

	c := &conn2{
		inbox:    make(chan network.Stream),
		host:     host,
		disc:     disc,
		cancel:   cancel,
		sessions: make(map[peer.ID]*session),
	}
	// TODO use different protocols for separating the session init from adding channels?
	host.SetStreamHandler(protocolID, c.handleStream)
	return c, nil
}

func (c *conn2) Close() error {
	// XXX wait for advertiser to shut down?
	close(c.inbox)
	c.cancel()
	return errorsJoin(
		c.disc.Close(),
		c.host.Close(),
	)
}

func (c *conn2) handleStream(stream network.Stream) {
	peer := stream.Conn().RemotePeer()
	c.mu.Lock()
	session := c.sessions[peer]
	c.mu.Unlock()
	if session == nil {
		c.inbox <- stream
	} else {
		session.inbox <- stream
	}
}

func (c *conn2) Accept() (mux.Session, error) {
	stream, ok := <-c.inbox
	if !ok {
		return nil, fmt.Errorf("closed")
	}
	peer := stream.Conn().RemotePeer()
	session := &session{
		host:  c.host,
		peer:  peer,
		inbox: make(chan network.Stream),
	}
	c.mu.Lock()
	c.sessions[peer] = session
	c.mu.Unlock()
	// TODO keep this channel open as a way to signal when the session is closed?
	if err := closeACK(stream); err != nil {
		return nil, err
	}
	return session, nil
}
