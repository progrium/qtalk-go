package libp2p

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/progrium/qtalk-go/mux"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/ipfs/go-log/v2"
)

const protocolID protocol.ID = "/qtalk/1.0.0"

var logger = log.Logger("rendezvous")

// Default addresses, should we limit our default to just QUIC?
// "/ip4/0.0.0.0/tcp/0",
// "/ip4/0.0.0.0/udp/0/quic",
// "/ip4/0.0.0.0/udp/0/quic-v1",
// "/ip4/0.0.0.0/udp/0/quic-v1/webtransport",
// "/ip6/::/tcp/0",
// "/ip6/::/udp/0/quic",
// "/ip6/::/udp/0/quic-v1",
// "/ip6/::/udp/0/quic-v1/webtransport",

type Conn interface {
	io.Closer
	Accept() (mux.Session, error)
}

type conn struct {
	host   myHost
	inbox  chan network.Stream
	disc   discoverer
	cancel context.CancelFunc
}

type chainedClose struct {
	io.ReadWriteCloser
	closer io.Closer
}

func (c chainedClose) Close() error {
	return errorsJoin(
		c.ReadWriteCloser.Close(),
		c.closer.Close(),
	)
}

func Dial(rendezvous string) (mux.Session, error) {
	dialTimeout := time.Second * 10
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	return DialContext(ctx, rendezvous)
}

func p2p(ctx context.Context) (myHost, discoverer, error) {
	host, err := libp2p.New()
	if err != nil {
		return nil, discoverer{}, err
	}
	disc, err := discover(ctx, host)
	if err != nil {
		return nil, discoverer{}, errorsJoin(err, host.Close())
	}
	return host, disc, nil
}

func DialContext(ctx context.Context, rendezvous string) (mux.Session, error) {
	host, stream, err := connectToPeer(ctx, rendezvous)
	if err != nil {
		return nil, err
	}
	return mux.New(chainedClose{stream, host}), nil
}

func connectToPeer(ctx context.Context, rendezvous string) (myHost, network.Stream, error) {
	host, disc, err := p2p(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer disc.Close()

	for {
		peerChan, err := disc.FindPeers(ctx, rendezvous)
		if err != nil {
			return nil, nil, errorsJoin(host.Close(), err)
		}

		for peer := range peerChan {
			if peer.ID == host.ID() || len(peer.Addrs) == 0 {
				continue
			}
			logger.Info("Found peer:", peer)

			logger.Info("Connecting to:", peer)
			stream, err := host.NewStream(ctx, peer.ID, protocolID)

			if err != nil {
				logger.Info("Connection failed:", err)
				continue
			}

			logger.Info("Finished connecting")

			return host, stream, nil
		}

		select {
		case <-ctx.Done():
			return nil, nil, errorsJoin(host.Close(), fmt.Errorf("unable to connect to peers"))
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func Listen(rendezvous string) (Conn, error) {
	ctx := context.Background()
	return ListenCtx(ctx, rendezvous)
}

func ListenCtx(ctx context.Context, rendezvous string) (Conn, error) {
	host, disc, err := p2p(ctx)
	if err != nil {
		return nil, err
	}

	ctx2, cancel := context.WithCancel(ctx)
	dutil.Advertise(ctx2, disc, rendezvous)

	c := &conn{
		inbox:  make(chan network.Stream),
		host:   host,
		disc:   disc,
		cancel: cancel,
	}
	host.SetStreamHandler(protocolID, c.handleStream)
	return c, nil
}

func (c *conn) Close() error {
	// XXX wait for advertiser to shut down?
	c.cancel()
	return errorsJoin(
		c.disc.Close(),
		c.host.Close(),
	)
}

func (c *conn) handleStream(stream network.Stream) {
	c.inbox <- stream
}

func (c *conn) Accept() (mux.Session, error) {
	// XXX cancel if the connection is closed
	s := <-c.inbox
	return mux.New(s), nil
}

type discoverer struct {
	io.Closer
	discovery.Discovery
}

func discover(ctx context.Context, host host.Host) (discoverer, error) {
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		return discoverer{}, err
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return discoverer{}, errorsJoin(err, kademliaDHT.Close())
	}

	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				logger.Warning(err)
			}
		}()
	}
	wg.Wait()
	// XXX check if we failed to connect to any bootstrap peers
	if ctx.Err() != nil {
		return discoverer{}, errorsJoin(ctx.Err(), kademliaDHT.Close())
	}

	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	return discoverer{
		kademliaDHT,
		routingDiscovery,
	}, nil
}
