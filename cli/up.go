package cli

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/hyprspace/hyprspace/p2p"
	"github.com/hyprspace/hyprspace/tun"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nxadm/tail"
)

var (
	// iface is the tun device used to pass packets between
	// Hyprspace and the user's machine.
	tunDev *tun.TUN
	// RevLookup allow quick lookups of an incoming stream
	// for security before accepting or responding to any data.
	RevLookup map[string]string
	// activeStreams is a map of active streams to a peer
	activeStreams map[string]network.Stream
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

	// Setup System Context
	ctx := context.Background()

	// Read in configuration from file.
	cfg, err := config.Read(configPath)
	checkErr(err)

	if cfg.Verbose {
		ctx = context.WithValue(ctx, config.WithVerbose, true)
	}

	if !flags.Foreground {
		if err := createDaemon(cfg); err != nil {
			fmt.Println("[+] Failed to Create Hyprspace Daemon")
			fmt.Println(err)
		} else {
			fmt.Println("[+] Successfully Created Hyprspace Daemon")
		}
		return
	}

	// Setup reverse lookup hash map for authentication.
	RevLookup = make(map[string]string, len(cfg.Peers))
	for ip, id := range cfg.Peers {
		RevLookup[id.ID] = ip
	}

	fmt.Println("[+] Creating TUN Device")

	if runtime.GOOS == "darwin" {
		if len(cfg.Peers) > 1 {
			checkErr(errors.New("cannot create interface macos does not support more than one peer"))
		}

		// Grab ip address of only peer in config
		var destPeer string
		for ip := range cfg.Peers {
			destPeer = ip
		}

		// Create new TUN device
		tunDev, err = tun.New(
			cfg.Interface.Name,
			tun.Address(cfg.Interface.Address),
			tun.DestAddress(destPeer),
			tun.MTU(1420),
		)
	} else {
		// Create new TUN device
		tunDev, err = tun.New(
			cfg.Interface.Name,
			tun.Address(cfg.Interface.Address),
			tun.MTU(1420),
		)
	}
	if err != nil {
		checkErr(err)
	}

	fmt.Println("[+] Creating LibP2P Node")

	// Check that the listener port is available.
	port, err := verifyPort(cfg.Interface.ListenPort)
	checkErr(err)

	// Create P2P Node
	host, dht, err := p2p.CreateNode(
		ctx,
		cfg.Interface.PrivateKey,
		port,
		streamHandler,
	)
	checkErr(err)

	if cfg.Verbose {
		go p2p.DebugEvents(host, dht)
	}

	// Setup Peer Table for Quick Packet --> Dest ID lookup
	peerTable := make(map[string]peer.ID)
	for ip, id := range cfg.Peers {
		peerTable[ip], err = peer.Decode(id.ID)
		checkErr(err)
	}

	fmt.Println("[+] Setting Up Node Discovery via DHT")

	// Setup P2P Discovery
	go p2p.Discover(ctx, host, dht, peerTable, cfg.Interface.Name)
	go prettyDiscovery(ctx, host, peerTable)

	// Configure path for lock
	lockPath := filepath.Join(filepath.Dir(cfg.Path), cfg.Interface.Name+".lock")

	// Register the application to listen for SIGINT/SIGTERM
	go signalExit(host, lockPath)

	// Write lock to filesystem to indicate an existing running daemon.
	err = os.WriteFile(lockPath, []byte(fmt.Sprint(os.Getpid())), os.ModePerm)
	checkErr(err)

	// Bring Up TUN Device
	err = tunDev.Up()
	if err != nil {
		checkErr(errors.New("unable to bring up tun device"))
	}

	fmt.Println("[+] Network Setup Complete...Waiting on Node Discovery")

	// + ----------------------------------------+
	// | Listen For New Packets on TUN Interface |
	// + ----------------------------------------+

	// Initialize active streams map and packet byte array.
	activeStreams = make(map[string]network.Stream)
	var packet = make([]byte, 1420)
	for {
		// Read in a packet from the tun device.
		plen, err := tunDev.Iface.Read(packet)
		if err != nil {
			log.Println(err)
			continue
		}

		// Decode the packet's destination address
		dst := net.IPv4(packet[16], packet[17], packet[18], packet[19]).String()

		// Check if we already have an open connection to the destination peer.
		stream, ok := activeStreams[dst]
		if ok {
			// Write out the packet's length to the libp2p stream to ensure
			// we know the full size of the packet at the other end.
			err = binary.Write(stream, binary.LittleEndian, uint16(plen))
			if err == nil {
				// Write the packet out to the libp2p stream.
				// If everyting succeeds continue on to the next packet.
				_, err = stream.Write(packet[:plen])
				if err == nil {
					continue
				}
			}
			// If we encounter an error when writing to a stream we should
			// close that stream and delete it from the active stream map.
			stream.Close()
			delete(activeStreams, dst)
		}

		// Check if the destination of the packet is a known peer to
		// the interface.
		if peer, ok := peerTable[dst]; ok {
			stream, err = host.NewStream(ctx, peer, p2p.Protocol)
			if err != nil {
				continue
			}
			// Write packet length
			err = binary.Write(stream, binary.LittleEndian, uint16(plen))
			if err != nil {
				stream.Close()
				continue
			}
			// Write the packet
			_, err = stream.Write(packet[:plen])
			if err != nil {
				stream.Close()
				continue
			}

			// If all succeeds when writing the packet to the stream
			// we should reuse this stream by adding it active streams map.
			activeStreams[dst] = stream
		}
	}
}

// singalExit registers two syscall handlers on the system  so that if
// an SIGINT or SIGTERM occur on the system hyprspace can gracefully
// shutdown and remove the filesystem lock file.
func signalExit(host host.Host, lockPath string) {
	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	// Shut the node down
	err := host.Close()
	checkErr(err)

	// Remove daemon lock from file system.
	err = os.Remove(lockPath)
	checkErr(err)

	fmt.Println("Received signal, shutting down...")

	// Exit the application.
	os.Exit(0)
}

// createDaemon handles creating an independent background process for a
// Hyprspace daemon from the original parent process.
func createDaemon(cfg *config.Config) error {
	path, err := os.Executable()
	checkErr(err)

	// Generate log path
	logPath := filepath.Join(filepath.Dir(cfg.Path), cfg.Interface.Name+".log")

	// Create Pipe to monitor for daemon output.
	f, err := os.Create(logPath)
	checkErr(err)

	// Create Sub Process
	process, err := os.StartProcess(
		path,
		append(os.Args, "--foreground"),
		&os.ProcAttr{
			Dir:   ".",
			Env:   os.Environ(),
			Files: []*os.File{nil, f, f},
		},
	)
	checkErr(err)

	// Listen to the child process's log output to determine
	// when the daemon is setup and connected to a set of peers.
	count := 0
	deadlineHit := false
	countChan := make(chan int)
	go func(out chan<- int) {
		numConnected := 0
		t, err := tail.TailFile(logPath, tail.Config{Follow: true})
		if err != nil {
			out <- numConnected
			return
		}
		for line := range t.Lines {
			fmt.Println(line.Text)
			if strings.HasPrefix(line.Text, "[+] Connection to") {
				numConnected++
				if numConnected >= len(cfg.Peers) {
					break
				}
			}
		}
		out <- numConnected
	}(countChan)

	// Block until all clients are connected or for a maximum of 30s.
	select {
	case _, deadlineHit = <-time.After(30 * time.Second):
	case count = <-countChan:
	}

	// Release the created daemon
	err = process.Release()
	checkErr(err)

	// Check if the daemon exited prematurely
	if !deadlineHit && count < len(cfg.Peers) {
		return errors.New("failed to create daemon")
	}
	return nil
}

func streamHandler(stream network.Stream) {
	// If the remote node ID isn't in the list of known nodes don't respond.
	if _, ok := RevLookup[stream.Conn().RemotePeer().Pretty()]; !ok {
		stream.Reset()
		return
	}
	var packet = make([]byte, 1420)
	var packetSize = make([]byte, 2)
	for {
		// Read the incoming packet's size as a binary value.
		_, err := stream.Read(packetSize)
		if err != nil {
			stream.Close()
			return
		}

		// Decode the incoming packet's size from binary.
		size := binary.LittleEndian.Uint16(packetSize)

		// Read in the packet until completion.
		var plen uint16 = 0
		for plen < size {
			tmp, err := stream.Read(packet[plen:size])
			plen += uint16(tmp)
			if err != nil {
				stream.Close()
				return
			}
		}
		tunDev.Iface.Write(packet[:size])
	}
}

func prettyDiscovery(ctx context.Context, node host.Host, peerTable map[string]peer.ID) {
	// Build a temporary map of peers to limit querying to only those
	// not connected.
	tempTable := make(map[string]peer.ID, len(peerTable))
	for ip, id := range peerTable {
		tempTable[ip] = id
	}
	for len(tempTable) > 0 {
		for ip, id := range tempTable {
			stream, err := node.NewStream(ctx, id, p2p.Protocol)
			if err != nil && (strings.HasPrefix(err.Error(), "failed to dial") ||
				strings.HasPrefix(err.Error(), "no addresses")) {
				// Attempt to connect to peers slowly when they aren't found.
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

func verifyPort(port int) (int, error) {
	var ln net.Listener
	var err error

	// If a user manually sets a port don't try to automatically
	// find an open port.
	if port != 8001 {
		ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			return port, errors.New("could not create node, listen port already in use by something else")
		}
	} else {
		// Automatically look for an open port when a custom port isn't
		// selected by a user.
		for {
			ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
			if err == nil {
				break
			}
			if port >= 65535 {
				return port, errors.New("failed to find open port")
			}
			port++
		}
	}
	if ln != nil {
		ln.Close()
	}
	return port, nil
}
