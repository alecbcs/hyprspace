package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
)

type NetworkPeer struct {
	IPaddr     string
	Id         string
	PeerID     peer.ID
	WriteChan  chan []byte //data written to peer
	ReadChan   chan []byte //data read from peer
	StreamChan chan network.Stream
}

// Discover starts up a DHT based discovery system finding and adding nodes with the same rendezvous string.
// The rendesvous string is necessary for distinguishing which network a peer is on.
func Discover(ctx context.Context, node *Libp2pNode, rendezvous string, peerTable map[string]*NetworkPeer) {
	node.Discovery.Advertise(ctx, rendezvous)

	// We want to be able to lookup a peer by it's ID
	peersByID := make(map[string]*NetworkPeer)
	for _, p := range peerTable {
		peersByID[p.Id] = p
	}

	// We need to monitor reachability here. If our node is not reachable and therefore has a
	// relay listen addr, and we have another peer who is publically reachable we want
	// to dial them and have a direct connection, rather than them dialing us over the relay.
	// I think this wont be necessary when p2p-circuit v2 is implemented.
	var reachable bool = true

	// Do it once before the timer starts
	findConnect(ctx, node, rendezvous, peersByID, reachable)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	republishTicker := time.NewTicker(time.Hour * 3)
	defer republishTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-node.SubReachability.Out():
			if !ok {
				return
			}
			evt, ok := ev.(event.EvtLocalReachabilityChanged)
			if !ok {
				return
			}
			if evt.Reachability == network.ReachabilityPublic {
				reachable = true
			} else {
				reachable = false
			}
		case <-republishTicker.C:
			node.Discovery.Advertise(ctx, rendezvous)
		case <-ticker.C:
			findConnect(ctx, node, rendezvous, peersByID, reachable)
		}
	}
}

// Finds peers providing the rendesvous string and initiates a hyprspace connection with them
func findConnect(ctx context.Context, node *Libp2pNode, rendezvous string, peerTable map[string]*NetworkPeer, reachable bool) {
	peers, err := discovery.FindPeers(ctx, node.Discovery, rendezvous)
	if err != nil {
		fmt.Println(err)
	}

	for _, p := range peers {
		networkPeer := peerTable[p.ID.Pretty()]
		if networkPeer == nil {
			continue
		}

		if node.Host.Network().Connectedness(p.ID) != network.Connected {
			// Dont dial this peer if we are reachable and it is listening on a relay
			if peerHasRelay(p) && reachable {
				continue
			}
			_, err := node.Host.Network().DialPeer(ctx, p.ID)

			// Dont connect to peer if there is already an open hyprspace stream
			// This somewhat prevents scenario where two peers dial each other simultaneously.
			// If we get two streams its not the end of the world, though
			if err != nil || peerHasStreams(node, networkPeer) {
				fmt.Println("Error dialing:", err)
				continue
			}
			stream, err := node.Host.NewStream(ctx, p.ID, Protocol)
			if err != nil {
				fmt.Println("Error opening stream:", err)
				continue
			}
			stream.Write([]byte{}) //this has to happen apparently for the streamhandler to be called on the other side
			networkPeer.StreamChan <- stream
			go ReadFromPeer(stream, networkPeer)
		}
	}
}

// Reads data from the stream and sends it to p.ReadChan
func ReadFromPeer(stream network.Stream, p *NetworkPeer) {
	var bytes []byte = make([]byte, 4000)

	for {
		i, err := stream.Read(bytes[:])

		if err != nil {
			break
		}
		if len(bytes) > 0 {
			p.ReadChan <- bytes[:i]
		}
	}
	stream.Reset()
	fmt.Println("[-]", p.IPaddr, "disconnected")
}

// Returns true if a hyprspace stream is already open with a peer
func peerHasStreams(node *Libp2pNode, networkPeer *NetworkPeer) (hasStreams bool) {
	conns := node.Host.Network().ConnsToPeer(networkPeer.PeerID)
	for _, con := range conns {
		for _, stream := range con.GetStreams() {
			if stream.Protocol() == Protocol {
				return true
			}
		}
	}
	return false
}

func peerHasRelay(p peer.AddrInfo) bool {
	for _, ma := range p.Addrs {
		if strings.HasSuffix(ma.String(), "p2p-circuit") {
			return true
		}
	}
	return false
}
