package p2p

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Discover starts up a DHT based discovery system finding and adding nodes with the same rendezvous string.
// The rendesvous string is necessary for distinguishing which network a peer is on.
func Discover(ctx context.Context, h *Hyprspace, node *Libp2pNode, rendezvous string, peerTable map[string]*NetworkPeer) {
	// Be sneaky and hash our discovery topic
	topicSHA256 := sha256.Sum256([]byte("hyprspace-" + rendezvous))
	topic, err := node.PubSub.Join(hex.EncodeToString(topicSHA256[:]))
	if err != nil {
		fmt.Println("Failed to join pubsub topic", err)
	}
	defer topic.Close()

	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Failed to subscribe to topic", err)
	}

	// Start a routine to read from the topic and send them to a channel
	message := make(chan *pubsub.Message)
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				fmt.Println(err)
				break
			}
			message <- msg
		}
	}()

	// We want to be able to lookup a peer by it's ID
	peersByID := make(map[string]*NetworkPeer)
	for _, p := range peerTable {
		peersByID[p.Id] = p

		// Protect peers from being pruned
		node.Host.ConnManager().Protect(peer.ID(p.Id), "networkPeer")
	}

	// Do it once before the timer starts
	topic.Publish(ctx, []byte{0})

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			topic.Publish(ctx, []byte{0})
		case msg := <-message:
			connect(ctx, h, msg.ReceivedFrom, peersByID[msg.ReceivedFrom.Pretty()], node)
		}
	}
}

// Connects to a network peer and opens a hyprspace stream if there is not one already
func connect(ctx context.Context, h *Hyprspace, p peer.ID, networkPeer *NetworkPeer, node *Libp2pNode) {
	if networkPeer == nil || PeerHasStreams(node, p) {
		return
	}

	if node.Host.Network().Connectedness(p) != network.Connected {
		addrInfo, err := node.KadDHT.FindPeer(ctx, p)
		if err != nil {
			fmt.Println("Error finding peer:", err)
			return
		}

		// Dont dial this peer if we are reachable and it is listening on a relay
		// if peerHasRelay(node.Host.Peerstore().PeerInfo(p)) && reachability.Get() == network.ReachabilityPublic {
		// 	return
		// }

		err = node.Host.Connect(ctx, addrInfo)
		if err != nil {
			fmt.Println("Error dialing:", err)
			return
		}
	}

	// Dont connect to peer if there is already an open hyprspace stream
	// This somewhat prevents scenario where two peers dial each other simultaneously.
	// If we get two streams its not the end of the world, though
	if PeerHasStreams(node, p) {

		return
	}
	stream, err := node.Host.NewStream(ctx, p, Protocol)
	if err != nil {
		return
	}
	stream.Write([]byte{}) //this has to happen apparently for the streamhandler to be called on the other side

	// Start read routine
	go ReadFromPeer(h, stream, networkPeer.IPaddr)
}

// Returns true if a hyprspace stream is already open with a peer
func PeerHasStreams(node *Libp2pNode, p peer.ID) (hasStreams bool) {
	conns := node.Host.Network().ConnsToPeer(p)
	for _, con := range conns {
		for _, stream := range con.GetStreams() {
			if stream.Protocol() == Protocol {
				return true
			}
		}
	}
	return false
}

// Returns true if there is a p2p-circuit address in the peer.AddrInfo
func peerHasRelay(p peer.AddrInfo) bool {
	for _, ma := range p.Addrs {
		if strings.HasSuffix(ma.String(), "p2p-circuit") {
			return true
		}
	}
	return false
}
