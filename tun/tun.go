package tun

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

// New creates and returns a new TUN interface for the application.
func New(name string) (result *water.Interface, err error) {
	// Setup TUN Config
	cfg := water.Config{
		DeviceType: water.TUN,
	}
	cfg.Name = name
	// Create TUN Interface
	result, err = water.New(cfg)
	return
}

// SetMTU sets the Maximum Tansmission Unit Size for a
// Packet on the interface.
func SetMTU(name string, mtu int) (err error) {
	return ip("link", "set", "dev", name, "mtu", fmt.Sprintf("%d", mtu))
}

// SetAddress sets the interface's known address and subnet.
func SetAddress(name string, address string) (err error) {
	return ip("addr", "add", address, "dev", name)
}

// Up brings up an interface to allow it to start accepting connections.
func Up(name string) (err error) {
	return ip("link", "set", "dev", name, "up")
}

// Down brings down an interface stopping active connections.
func Down(name string) (err error) {
	return ip("link", "set", "dev", name, "down")
}

// Delete removes a TUN device from the host.
func Delete(name string) (err error) {
	return ip("link", "delete", "dev", name)
}

func ip(args ...string) (err error) {
	cmd := exec.Command("/sbin/ip", args...)
	err = cmd.Run()
	return
}
