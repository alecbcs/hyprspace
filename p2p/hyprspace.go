package p2p

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/hyprspace/hyprspace/config"
	"github.com/hyprspace/hyprspace/tun"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/songgao/water"
)

// Protocol is a descriptor for the Hyprspace P2P Protocol.
const Protocol = "/hyprspace/0.0.3"

// Represents a hyprspace node and interface
type Hyprspace struct {
	// Name of the interface
	Name string
	// Libp2p node for the interface
	Node *Libp2pNode
	// iface is the tun device used to pass packets between
	// Hyprspace and the user's machine.
	iface *water.Interface
	// Interface write channel
	ifaceWrite chan []byte
	// Map of ip address to network peers
	PeerTable map[string]*NetworkPeer
	// Map of peer id to network peer ip addrs
	RevLookup map[string]string
	// Configuration for the interface
	Global config.Config
	// Context
	Ctx context.Context
}

// Represents a peer on the hyprspace interface
type NetworkPeer struct {
	// Peer's ipv4 addr as a string
	IPaddr string
	// Libp2p peer id as a string
	Id string
	// Libp2p peer id as a peer.ID (string)
	PeerID peer.ID
	// Channel used for writing data to peer
	WriteChan chan []byte
	// Channel used by discovery & streamhandler to send new streams to IO handler routines
	StreamChan chan network.Stream
}

var (
	// Used by the streamhandler
	// Maps our peer id to the hyprspace struct for that network
	hyprspace map[string]*Hyprspace
)

func init() {
	hyprspace = make(map[string]*Hyprspace)
}

// Start a hyprspace node and return it
func Up(interfaceName string, configPath string) (h *Hyprspace, err error) {
	h = new(Hyprspace)
	// Set the name
	h.Name = interfaceName
	// Setup the interface write channel
	h.ifaceWrite = make(chan []byte)

	if configPath == "" {
		configPath = "/etc/hyprspace/" + interfaceName + ".yaml"
	}

	// Read in configuration from file.
	h.Global, err = config.Read(configPath)
	if err != nil {
		return
	}

	// Check if this interface is already running
	if hyprspace[h.Global.Interface.ID] != nil {
		return nil, errors.New("Interface already running")
	}
	hyprspace[h.Global.Interface.ID] = h

	// Setup Peer Table
	h.PeerTable = make(map[string]*NetworkPeer, len(h.Global.Peers))

	// Setup reverse lookup hash map for authentication.
	h.RevLookup = make(map[string]string, len(h.Global.Peers))
	for ip, id := range h.Global.Peers {
		h.RevLookup[id.ID] = ip
	}

	// Create new TUN device
	h.iface, err = tun.New(h.Global.Interface.Name)
	if err != nil {
		fmt.Println("Failed to create tun interface:", err)
		return
	}
	// Set TUN MTU
	tun.SetMTU(h.Global.Interface.Name, 1420)
	// Add Address to Interface
	tun.SetAddress(h.Global.Interface.Name, h.Global.Interface.Address)
	// Bring Up TUN Device
	tun.Up(h.Global.Interface.Name)

	// Setup System Context
	h.Ctx = context.Background()

	// Create P2P Node
	fmt.Println("[+] Starting interface", h.Name)
	h.Node, err = CreateNode(h.Ctx, h.Global.Interface)

	// Setup peerTable and start goroutines for handling peer IO
	// peerTable maps an ip address string to a peer
	for ip, id := range h.Global.Peers {
		np := new(NetworkPeer)
		np.IPaddr = ip
		np.Id = id.ID
		np.PeerID, err = peer.Decode(id.ID)
		if err != nil {
			return
		}
		np.WriteChan = make(chan []byte)
		np.StreamChan = make(chan network.Stream)
		h.PeerTable[ip] = np
		go handlePeerIO(h, np)
	}

	// Setup Hyprspace Stream Handler
	h.Node.Host.SetStreamHandler(Protocol, streamHandler)

	// Setup P2P Discovery
	go Discover(h.Ctx, h, h.Node, h.Global.Interface.DiscoverKey, h.PeerTable)

	go interfaceListen(h)
	go interfaceWrite(h)
	return
}

func (h *Hyprspace) Shutdown() error {
	// Delete the tun device
	err := tun.Delete(h.Name)
	// Close the libp2p node
	h.Node.Host.Close()
	// Stop interfaceWrite routine
	close(h.ifaceWrite)
	// Stop IO goroutines by sending nil to all peers' StreamChan
	for _, np := range h.PeerTable {
		np.StreamChan <- nil
	}
	// Remove this interface from the map
	delete(hyprspace, h.Node.Host.ID().Pretty())
	return err
}

// Writes packets from the ifaceWrite channel to the tun interface
func interfaceWrite(h *Hyprspace) {
	for {
		select {
		case bytes, ok := <-h.ifaceWrite:
			if ok {
				h.iface.Write(bytes)
			} else {
				return
			}
		}
	}
}

// Listen for packets on TUN interface and send them to the correct peers
func interfaceListen(h *Hyprspace) {
	packet := make([]byte, 1420)

	// Set up a quicker LUT using a uint32 to map ipv4 to peer channel
	ipToPeer := make(map[uint32]chan []byte)
	for ip, peer := range h.PeerTable {
		ipBytes := net.ParseIP(ip).To4() //Must be converted to 4 byte representation

		// Represent ipv4 address as a 32 bit uint
		var intRep uint32 = (uint32(ipBytes[0]) << 24) | (uint32(ipBytes[1]) << 16) | (uint32(ipBytes[2]) << 8) | uint32(ipBytes[3])

		// Alert if ip is used more than once
		if _, exists := ipToPeer[intRep]; exists {
			fmt.Println("Routing table error: peer", ip, "is defined more than once")
		}
		ipToPeer[intRep] = peer.WriteChan
	}

	// Read packets and send them to the correct peers
	for {
		plen, err := h.iface.Read(packet)
		if err != nil {
			break
		}

		// Get destination ip from ip header and find peer to send it to
		ip := (uint32(packet[16]) << 24) | (uint32(packet[17]) << 16) | (uint32(packet[18]) << 8) | uint32(packet[19])

		p, ok := ipToPeer[ip]
		if ok {
			p <- packet[:plen]
		}
	}
}

// Should be started as a go routine for each peer
func handlePeerIO(h *Hyprspace, p *NetworkPeer) {
	var stream network.Stream = nil
	var size [2]byte

	for {
		select {
		case stream = <-p.StreamChan:
			if stream == nil {
				return // Shutdown
			}
			fmt.Println("[+]", h.Name, p.IPaddr, "connected")
		case bytes, ok := <-p.WriteChan:
			if ok && stream != nil {
				binary.BigEndian.PutUint16(size[:], uint16(len(bytes)))
				stream.Write(size[:])
				stream.Write(bytes[:])
			}
		}
	}
}

// Reads packets from the stream and sends them to the interface write channel
func ReadFromPeer(h *Hyprspace, stream network.Stream, ip string) {
	networkPeer, ok := h.PeerTable[ip]
	if !ok {
		fmt.Println("PeerTable lookup failed")
		return
	}

	// Send the stream to the IO handling routine
	networkPeer.StreamChan <- stream
	// Read from the peer and send received bytes to the readChan
	for {
		size, err := getBytes(stream, 2)
		if err != nil {
			break
		}
		packet, err := getBytes(stream, binary.BigEndian.Uint16(size))
		if err != nil {
			break
		}
		h.ifaceWrite <- packet[:]
	}
	stream.Reset()
	fmt.Println("[-]", h.Name, ip, "disconnected")
}

// Gets an exact number of bytes from a network stream or returns an error if the stream cannot be read.
func getBytes(stream network.Stream, i uint16) (bytes []byte, err error) {
	if i < 1 {
		return
	}
	bytes = make([]byte, i)
	var single []byte = make([]byte, 1)
	var total uint16 = 0
	count, err := stream.Read(single)
	for err == nil {
		if count > 0 {
			bytes[total] = single[0]
			total++
			if total == i {
				return
			}
		}
		count, err = stream.Read(single)
	}
	return
}

// Handles incoming hyprspace streams
func streamHandler(stream network.Stream) {
	h, ok := hyprspace[stream.Conn().LocalPeer().Pretty()]
	if !ok {
		// We shouldn't ever get here but just to be safe
		fmt.Println("No hyprspace instance found for this incoming stream?")
		return
	}

	// If the remote node ID isn't in the list of known nodes don't respond.
	ip, ok := h.RevLookup[stream.Conn().RemotePeer().Pretty()]
	if !ok {
		fmt.Println("Invalid peer", stream.Conn().RemotePeer().Pretty())
		stream.Reset()
		return
	}

	// Start routine to read from peer
	go ReadFromPeer(h, stream, ip)
}
