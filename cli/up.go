package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/hyprspace/hyprspace/p2p"
	"github.com/hyprspace/hyprspace/tun"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
)

var (
	// Global is the global interface configuration for the
	// application instance.
	Global config.Config
	// iface is the tun device used to pass packets between
	// Hyprspace and the user's machine.
	iface *water.Interface
	// RevLookup allow quick lookups of an incoming stream
	// for security before accepting or responding to any data.
	RevLookup map[string]string

	//Peer info table
	PeerTable    map[string]*p2p.NetworkPeer
	PeerTablePtr *map[string]*p2p.NetworkPeer
)

// Up creates and brings up a Hyprspace Interface.
var Up = cmd.Sub{
	Name:  "up",
	Alias: "up",
	Short: "Create and Bring Up a Hyprspace Interface.",
	Args:  &UpArgs{},
	Flags: &UpFlags{},
	Run:   UpRun,
}

// UpArgs handles the specific arguments for the up command.
type UpArgs struct {
	InterfaceName string
}

// UpFlags handles the specific flags for the up command.
type UpFlags struct {
	Foreground bool `short:"f" long:"foreground" desc:"Don't Create Background Daemon."`
}

// UpRun handles the execution of the up command.
func UpRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*UpArgs)

	// Parse Command Flags
	flags := c.Flags.(*UpFlags)

	// Parse Global Config Flag for Custom Config Path
	configPath := r.Flags.(*GlobalFlags).Config
	if configPath == "" {
		configPath = "/etc/hyprspace/" + args.InterfaceName + ".yaml"
	}

	// Read in configuration from file.
	Global, err := config.Read(configPath)
	checkErr(err)

	if !flags.Foreground {
		// Make results chan
		out := make(chan error)
		go createDaemon(out)

		select {
		case err = <-out:
		case <-time.After(30 * time.Second):
		}
		if err != nil {
			fmt.Println("[+] Failed to Create Hyprspace Daemon")
		} else {
			fmt.Println("[+] Successfully Created Hyprspace Daemon")
		}
		return
	}

	// Setup Peer Table
	PeerTable := make(map[string]*p2p.NetworkPeer, len(Global.Peers))
	PeerTablePtr = &PeerTable
	// Setup reverse lookup hash map for authentication.
	RevLookup = make(map[string]string, len(Global.Peers))
	for ip, id := range Global.Peers {
		RevLookup[id.ID] = ip
	}

	fmt.Println("[+] Creating TUN Device")
	// Create new TUN device
	iface, err = tun.New(Global.Interface.Name)
	if err != nil {
		checkErr(errors.New("interface already in use"))
	}
	// Set TUN MTU
	tun.SetMTU(Global.Interface.Name, 1420)
	// Add Address to Interface
	tun.SetAddress(Global.Interface.Name, Global.Interface.Address)

	// Setup System Context
	ctx := context.Background()

	fmt.Println("[+] Creating LibP2P Node")

	port := Global.Interface.ListenPort

	// Create P2P Node
	node, err := p2p.CreateNode(ctx,
		Global.Interface.PrivateKey,
		port)
	checkErr(err)

	// Setup peerTable and start goroutines for handling peer IO
	// peerTable maps an ip address string to a peer
	for ip, id := range Global.Peers {
		np := new(p2p.NetworkPeer)
		np.IPaddr = ip
		np.Id = id.ID
		np.PeerID = peer.ID(id.ID)
		np.WriteChan = make(chan []byte)
		np.ReadChan = make(chan []byte)
		np.StreamChan = make(chan network.Stream) //stream is established by discovery routine
		PeerTable[ip] = np
		go handlePeerIO(np)
	}

	// Setup Hyprspace Stream Handler
	node.Host.SetStreamHandler(p2p.Protocol, streamHandler)

	fmt.Println("[+] Setting Up Node Discovery via DHT")
	// Setup P2P Discovery
	go p2p.Discover(ctx, node, Global.Interface.DiscoverKey, PeerTable)
	//go prettyDiscovery(ctx, host, PeerTable)

	go func() {
		// Wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("Received signal, shutting down...")

		// Shut the node down
		if err := node.Host.Close(); err != nil {
			panic(err)
		}
		os.Exit(0)
	}()

	// Bring Up TUN Device
	tun.Up(Global.Interface.Name)

	fmt.Println("[+] Network Setup Complete...Waiting on Node Discovery")
	// Listen For New Packets on TUN Interface
	packet := make([]byte, 1420)
	var header *ipv4.Header
	var plen int
	//read packets and send them to peer channels
	for {
		plen, err = iface.Read(packet)
		checkErr(err)
		header, _ = ipv4.ParseHeader(packet)

		p, ok := PeerTable[header.Dst.String()]
		if ok {
			p.WriteChan <- packet[:plen]
		}
	}
}

// Should be started as a go routine for each peer
func handlePeerIO(p *p2p.NetworkPeer) {
	var stream network.Stream = nil

	for {
		select {
		case bytes, ok := <-p.WriteChan:
			if ok && stream != nil {
				stream.Write(bytes[:])
			}
		case bytes, ok := <-p.ReadChan:
			if ok && stream != nil {
				iface.Write(bytes) //write packet from peer to network interface
			}
		case stream = <-p.StreamChan:
			fmt.Println("[+]", p.IPaddr, "connected")
		}
	}
}

func streamHandler(stream network.Stream) {
	// If the remote node ID isn't in the list of known nodes don't respond.
	ip, ok := RevLookup[stream.Conn().RemotePeer().Pretty()]
	if !ok {
		fmt.Println("Invalid peer", stream.Conn().RemotePeer().Pretty())
		stream.Reset()
		return
	}
	networkPeer := (*PeerTablePtr)[ip]

	// Send the stream
	networkPeer.StreamChan <- stream

	go p2p.ReadFromPeer(stream, networkPeer)
}

func createDaemon(out chan<- error) {
	path, err := os.Executable()
	checkErr(err)

	// Create Sub Process
	process, err := os.StartProcess(
		path,
		append(os.Args, "--foreground"),
		&os.ProcAttr{
			Files: []*os.File{nil, nil, nil},
		},
	)
	checkErr(err)

	process.Release()
	out <- nil
}
