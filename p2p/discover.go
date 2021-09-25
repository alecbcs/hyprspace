package p2p

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type NetworkPeer struct {
	IPaddr     string
	Id         string
	PeerID     peer.ID
	WriteChan  chan []byte //data written to peer
	ReadChan   chan []byte //data read from peer
	StreamChan chan network.Stream
}

type Reachability struct {
	mu           sync.RWMutex
	reachability network.Reachability
}

func (r *Reachability) Set(data network.Reachability) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reachability = data
}

func (r *Reachability) Get() network.Reachability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.reachability
}

// Discover starts up a DHT based discovery system finding and adding nodes with the same rendezvous string.
// The rendesvous string is necessary for distinguishing which network a peer is on.
func Discover(ctx context.Context, node *Libp2pNode, rendezvous string, peerTable map[string]*NetworkPeer) {
	topicSHA256 := sha256.Sum256([]byte(fmt.Sprintf("hyprspace-%s", rendezvous)))
	topic, err := node.PubSub.Join(hex.EncodeToString(topicSHA256[:]))
	if err != nil {
		fmt.Println("Failed to join pubsub topic", err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Failed to subscribe to topic", err)
	}

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
	}

	// We need to monitor reachability here. If our node is not reachable and therefore has a
	// relay listen addr, and we have another peer who is publically reachable we want
	// to dial them and have a direct connection, rather than them dialing us over the relay.
	// I think this wont be necessary when p2p-circuit v2 is implemented.
	var reachable Reachability

	// Do it once before the timer starts
	topic.Publish(ctx, []byte{0})

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

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
			reachable.Set(evt.Reachability)
		case <-ticker.C:
			topic.Publish(ctx, []byte{0})
		case msg := <-message:
			connect(ctx, msg.ReceivedFrom, peersByID[msg.ReceivedFrom.Pretty()], node, &reachable)
		}
	}
}

// Connects to a network peer and opens a hyprspace stream if there is not one already
func connect(ctx context.Context, p peer.ID, networkPeer *NetworkPeer, node *Libp2pNode, reachability *Reachability) {
	if networkPeer == nil || peerHasStreams(node, p) {
		return
	}

	if node.Host.Network().Connectedness(p) != network.Connected {
		addrInfo, err := node.KadDHT.FindPeer(ctx, p)
		if err != nil {
			fmt.Println("Error finding peer:", err)
			return
		}

		// Dont dial this peer if we are reachable and it is listening on a relay
		if peerHasRelay(node.Host.Peerstore().PeerInfo(p)) && reachability.Get() == network.ReachabilityPublic {
			return
		}

		err = node.Host.Connect(ctx, addrInfo)
		if err != nil {
			fmt.Println("Error dialing:", err)
			return
		}
	}

	// Dont connect to peer if there is already an open hyprspace stream
	// This somewhat prevents scenario where two peers dial each other simultaneously.
	// If we get two streams its not the end of the world, though
	if peerHasStreams(node, p) {
		return
	}
	stream, err := node.Host.NewStream(ctx, p, Protocol)
	if err != nil {
		fmt.Println("Error opening stream:", err)
		return
	}
	stream.Write([]byte{}) //this has to happen apparently for the streamhandler to be called on the other side

	// Send stream for IO handling and start read routine
	networkPeer.StreamChan <- stream
	go ReadFromPeer(stream, networkPeer)
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
		} else {
			time.Sleep(time.Millisecond * 1)
		}
	}
	stream.Reset()
	fmt.Println("[-]", p.IPaddr, "disconnected")
}

// Returns true if a hyprspace stream is already open with a peer
func peerHasStreams(node *Libp2pNode, p peer.ID) (hasStreams bool) {
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
