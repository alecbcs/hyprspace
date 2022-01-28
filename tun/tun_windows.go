//go:build windows
// +build windows

package tun

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/songgao/water"
)

// New creates and returns a new TUN interface for the application.
func New(name, address string) (result *water.Interface, err error) {
	// TUN on Windows requires address and network to be set on device creation stage
	// We also set network to 0.0.0.0/0 so we able to reach networks behind the node
	// https://github.com/songgao/water/blob/master/params_windows.go
	// https://gitlab.com/openconnect/openconnect/-/blob/master/tun-win32.c
	ip, _, err := net.ParseCIDR(address)
	if err != nil {
		return nil, err
	}
	network := net.IPNet{
		IP:   ip,
		Mask: net.IPv4Mask(0, 0, 0, 0),
	}
	// Setup TUN Config
	cfg := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID:   "tap0901",
			InterfaceName: name,
			Network:       network.String(),
		},
	}
	// Create TUN Interface
	result, err = water.New(cfg)
	return
}

// SetMTU sets the Maximum Tansmission Unit Size for a
// Packet on the interface.
func SetMTU(name string, mtu int) (err error) {
	return netsh("interface", "ipv4", "set", "subinterface", name, "mtu=", fmt.Sprintf("%d", mtu))
}

// SetAddress sets the interface's known address and subnet.
func SetAddress(name string, address string) (err error) {
	return netsh("interface", "ip", "set", "address", "name=", name, "static", address)
}

// Up brings up an interface to allow it to start accepting connections.
func Up(name string) (err error) {
	return
}

// Down brings down an interface stopping active connections.
func Down(name string) (err error) {
	return
}

// Delete removes a TUN device from the host.
func Delete(name string) (err error) {
	return
}

func netsh(args ...string) (err error) {
	cmd := exec.Command("netsh", args...)
	err = cmd.Run()
	return
}
