package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/hyprspace/hyprspace/config"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-tcp-transport"
)

type Libp2pNode struct {
	// Represents the host context layer
	Ctx context.Context

	// Represents the libp2p host
	Host host.Host

	// Represents the DHT routing table
	KadDHT *dht.IpfsDHT

	// Represents the peer discovery service
	Discovery *discovery.RoutingDiscovery

	// Represents the PubSub Handler
	PubSub *pubsub.PubSub

	// NAT reachability events
	//SubReachability event.Subscription
}

// Protocol is a descriptor for the Hyprspace P2P Protocol.
const Protocol = "/hyprspace/0.0.2"

// CreateNode creates an internal Libp2p nodes and returns it and it's DHT Discovery service.
func CreateNode(ctx context.Context, cfg config.Interface, port int) (node *Libp2pNode, err error) {
	node = new(Libp2pNode)

	// Unmarshal Private Key
	privateKey, err := crypto.UnmarshalPrivateKey([]byte(cfg.PrivateKey))
	if err != nil {
		return
	}

	// Setup connection manager
	connMgr := connmgr.NewConnManager(
		40,
		400,
		time.Minute*1,
	)

	var opts libp2p.Option
	for _, t := range cfg.Transports {
		if t == "quic" {
			opts = libp2p.ChainOptions(libp2p.Transport(libp2pquic.NewTransport), opts)
		}
		if t == "tcp" {
			opts = libp2p.ChainOptions(libp2p.Transport(tcp.NewTCPTransport), opts)
		}
	}

	if cfg.AutoRelay {
		opts = libp2p.ChainOptions(libp2p.EnableAutoRelay(), opts)
	}

	// Listen addresses
	ip6quic := fmt.Sprintf("/ip6/::/udp/%d/quic", port)
	ip4quic := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port)

	ip6tcp := fmt.Sprintf("/ip6/::/tcp/%d", port)
	ip4tcp := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)

	// Create libp2p node
	node.Host, err = libp2p.New(ctx,
		opts, // Config options
		libp2p.ListenAddrStrings(ip6quic, ip4quic, ip6tcp, ip4tcp),
		libp2p.Identity(privateKey),
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.ConnectionManager(connMgr),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			bootstrapPeers := dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...)
			dstore := dsync.MutexWrap(ds.NewMapDatastore())
			node.KadDHT, err = dht.New(ctx, h, bootstrapPeers, dht.Datastore(dstore), dht.Mode(dht.ModeAutoServer))

			node.KadDHT.Bootstrap(ctx)
			return node.KadDHT, err
		}),
		libp2p.FallbackDefaults,
	)
	if err != nil {
		return
	}

	// Setup reachability event chan
	//node.SubReachability, _ = node.Host.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))

	// Setup routing discovery
	node.Discovery = discovery.NewRoutingDiscovery(node.KadDHT)

	// Create a PubSub handler with the routing discovery
	node.PubSub, err = setupPubSub(ctx, node.Host, node.Discovery)

	return
}

// A function that generates a PubSub Handler object and returns it
// Requires a node host and a routing discovery service.
func setupPubSub(ctx context.Context, nodehost host.Host, routingdiscovery *discovery.RoutingDiscovery) (*pubsub.PubSub, error) {
	// Create a new PubSub service which uses a GossipSub router
	pubsubhandler, err := pubsub.NewGossipSub(ctx, nodehost, pubsub.WithDiscovery(routingdiscovery))
	// Handle any potential error
	if err != nil {
		return nil, err
	}

	// Return the PubSub handler
	return pubsubhandler, nil
}
