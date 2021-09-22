package p2p

import (
	"context"
	"fmt"
	"time"

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

	// Do it once before the timer starts
	findConnect(ctx, node, rendezvous, peersByID)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			findConnect(ctx, node, rendezvous, peersByID)
		}
	}
}

// Finds peers providing the rendesvous string and initiates a hyprspace connection with them
func findConnect(ctx context.Context, node *Libp2pNode, rendezvous string, peerTable map[string]*NetworkPeer) {
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
