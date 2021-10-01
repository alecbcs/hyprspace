package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"net/rpc"

	"github.com/hyprspace/hyprspace/p2p"
)

const PORT = 5882

type Daemon struct {
	// Interface name -> Hyprspace struct
	Interfaces map[string]*p2p.Hyprspace
	// Out chan for shutting down process
	Out      chan error
	Listener *net.Listener
}

var (
	daemon *Daemon
)

// Runs the daemon in it's own process
func StartDaemonProcess() (err error) {
	path, err := os.Executable()
	process, err := os.StartProcess(
		path,
		[]string{"hyprspace", "daemon", "up"},
		&os.ProcAttr{
			Files: []*os.File{nil, nil, nil},
		},
	)
	if err != nil {
		return
	}
	err = process.Release()
	return
}

// Starts the daemon in the current program
func Run(out chan error) {
	// Create a TCP listener that will listen on PORT
	listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(PORT))
	if err != nil {
		out <- err
		return
	}
	defer listener.Close()

	daemon = new(Daemon)
	daemon.Out = out
	daemon.Interfaces = make(map[string]*p2p.Hyprspace)
	daemon.Listener = &listener

	// Publish our Handler methods
	rpc.Register(&PeersHandler{D: daemon})
	rpc.Register(&ShutdownHandler{D: daemon})
	rpc.Register(&UpHandler{D: daemon})

	rpc.Accept(listener)
}

// Stops the daemon if it is running
func Shutdown() {
	if !isDaemonRunning() {
		return
	}
	// Make an rpc call to the daemon
	client, err := connectToDaemon()
	if err != nil {
		fmt.Println("Can't connect to daemon")
		return
	}
	defer client.Close()

	req := &ShutdownReq{Daemon: true}
	resp := new(ShutdownResp)
	_ = client.Call("ShutdownHandler.Execute", req, resp)
}

// Connects to the daemon and shuts down an interface
func DownInterface(iface string) (err error) {
	if !isDaemonRunning() {
		return
	}
	// Make an rpc call to the daemon
	client, err := connectToDaemon()
	if err != nil {
		return err
	}
	defer client.Close()

	req := &ShutdownReq{Daemon: false, Interface: iface}
	resp := new(ShutdownResp)
	err = client.Call("ShutdownHandler.Execute", req, resp)
	if err != nil {
		fmt.Println("Failed to call shutdown rpc", err)
		return err
	}
	return errors.New(resp.Error)
}

// Returns the peer ip addrs connected to an interface
func GetConnectedPeers(iface string) ([]string, error) {
	if !isDaemonRunning() {
		return nil, errors.New("Daemon not running")
	}
	// Make an rpc call to the daemon
	client, _ := connectToDaemon()
	defer client.Close()
	req := &PeersReq{Interface: iface}
	resp := new(PeersResp)
	err := client.Call("PeersHandler.Execute", req, resp)
	if err != nil {
		return nil, err
	}
	return resp.ConnectedPeers, errors.New(resp.Error)
}

// Returns true if the daemon is running
func isDaemonRunning() bool {
	listener, err := net.Listen("tcp", "localhost:"+strconv.Itoa(PORT))
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

// Brings an interface up
func UpInterface(iface string, configPath string) (err error) {
	if !isDaemonRunning() {
		err = StartDaemonProcess()
		if err != nil {
			return
		}
	}
	// Make an rpc call to the daemon
	var client *rpc.Client = nil
	for i := 0; i < 5; i++ {
		client, err = connectToDaemon()
		if err == nil {
			defer client.Close()
			break
		}
		time.Sleep(time.Second * 1)
	}
	if client == nil {
		return errors.New("Failed to connect to daemon")
	}

	req := &UpReq{Interface: iface, ConfigPath: configPath}
	resp := new(UpResp)
	err = client.Call("UpHandler.Execute", req, resp)
	if err != nil {
		return err
	}
	return errors.New(resp.Error)
}

func connectToDaemon() (client *rpc.Client, err error) {
	client, err = rpc.Dial("tcp", "localhost:"+strconv.Itoa(PORT))
	if err != nil {
		err = errors.New("Daemon is not running")
	}
	return
}
