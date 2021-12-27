package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/hyprspace/hyprspace/p2p"
	"github.com/hyprspace/hyprspace/tun"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
)

var (
	// iface is the tun device used to pass packets between
	// Hyprspace and the user's machine.
	iface *water.Interface
	// RevLookup allow quick lookups of an incoming stream
	// for security before accepting or responding to any data.
	RevLookup map[string]bool
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
	cfg, err := config.Read(configPath)
	checkErr(err)

	if !flags.Foreground {
		// Make results chan
		out := make(chan error)
		go createDaemon(cfg, out)

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

	// Setup reverse lookup hash map for authentication.
	RevLookup = make(map[string]bool, len(cfg.Peers))
	for _, id := range cfg.Peers {
		RevLookup[id.ID] = true
	}

	fmt.Println("[+] Creating TUN Device")
	// Create new TUN device
	iface, err = tun.New(cfg.Interface.Name, cfg.Interface.Address)
	if err != nil {
		checkErr(errors.New("interface already in use"))
	}
	// Set TUN MTU
	tun.SetMTU(cfg.Interface.Name, 1420)
	// Add Address to Interface
	tun.SetAddress(cfg.Interface.Name, cfg.Interface.Address)

	// Setup System Context
	ctx := context.Background()

	fmt.Println("[+] Creating LibP2P Node")

	// Check that the listener port is available.
	var ln net.Listener
	port := cfg.Interface.ListenPort
	if port != 8001 {
		ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			checkErr(errors.New("could not create node, listen port already in use by something else"))
		}
	} else {
		for {
			ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
			if err == nil {
				break
			}
			if port >= 65535 {
				checkErr(errors.New("failed to find open port"))
			}
			port++
		}
	}
	if ln != nil {
		ln.Close()
	}

	// Create P2P Node
	host, dht, err := p2p.CreateNode(ctx,
		cfg.Interface.PrivateKey,
		port,
		streamHandler)
	checkErr(err)

	// Setup Peer Table for Quick Packet --> Dest ID lookup
	peerTable := make(map[string]peer.ID)
	for ip, id := range cfg.Peers {
		peerTable[ip], err = peer.Decode(id.ID)
		checkErr(err)
	}

	fmt.Println("[+] Setting Up Node Discovery via DHT")
	// Setup P2P Discovery
	go p2p.Discover(ctx, host, dht, peerTable)
	go prettyDiscovery(ctx, host, peerTable)

	go func() {
		// Wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("Received signal, shutting down...")

		// Shut the node down
		if err := host.Close(); err != nil {
			panic(err)
		}
		os.Exit(0)
	}()

	// Bring Up TUN Device
	tun.Up(cfg.Interface.Name)

	fmt.Println("[+] Network Setup Complete...Waiting on Node Discovery")
	// Listen For New Packets on TUN Interface
	packet := make([]byte, 1420)
	var stream network.Stream
	var header *ipv4.Header
	var plen int
	for {
		plen, err = iface.Read(packet)
		checkErr(err)
		header, _ = ipv4.ParseHeader(packet)
		_, ok := cfg.Peers[header.Dst.String()]
		if ok {
			stream, err = host.NewStream(ctx, peerTable[header.Dst.String()], p2p.Protocol)
			if err != nil {
				log.Println(err)
				continue
			}
			stream.Write(packet[:plen])
			stream.Close()
		}
	}
}

func createDaemon(cfg config.Config, out chan<- error) {
	path, err := os.Executable()
	checkErr(err)
	// Create Pipe to monitor for daemon output.
	r, w, err := os.Pipe()
	checkErr(err)
	// Create Sub Process
	process, err := os.StartProcess(
		path,
		append(os.Args, "--foreground"),
		&os.ProcAttr{
			Files: []*os.File{nil, w, w},
		},
	)
	checkErr(err)
	scanner := bufio.NewScanner(r)
	count := 0
	for count < len(cfg.Peers) && scanner.Scan() {
		fmt.Println(scanner.Text())
		if strings.HasPrefix(scanner.Text(), "[+] Connection to") {
			count++
		}
	}
	err = process.Release()
	checkErr(err)
	if count < len(cfg.Peers) {
		out <- errors.New("failed to create daemon")
	}
	out <- nil
}

func streamHandler(stream network.Stream) {
	// If the remote node ID isn't in the list of known nodes don't respond.
	if _, ok := RevLookup[stream.Conn().RemotePeer().Pretty()]; !ok {
		stream.Reset()
	}
	io.Copy(iface.ReadWriteCloser, stream)
	stream.Close()
}

func prettyDiscovery(ctx context.Context, node host.Host, peerTable map[string]peer.ID) {
	tempTable := make(map[string]peer.ID, len(peerTable))
	for ip, id := range peerTable {
		tempTable[ip] = id
	}
	for len(tempTable) > 0 {
		for ip, id := range tempTable {
			stream, err := node.NewStream(ctx, id, p2p.Protocol)
			if err != nil && (strings.HasPrefix(err.Error(), "failed to dial") ||
				strings.HasPrefix(err.Error(), "no addresses")) {
				time.Sleep(5 * time.Second)
				continue
			}
			if err == nil {
				fmt.Printf("[+] Connection to %s Successful. Network Ready.\n", ip)
				stream.Close()
			}
			delete(tempTable, ip)
		}
	}
}
