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
func New(name string, opts ...Option) (*TUN, error) {
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

	// Create Water Interface
	iface, err := water.New(cfg)
	if err != nil {
		return nil, err
	}

	// Create TUN result struct
	result := TUN{
		Iface: iface,
	}

	// Apply options to set TUN config values
	err = result.Apply(opts...)
	return &result, err
}

// setMTU sets the Maximum Tansmission Unit Size for a
// Packet on the interface.
func (t *TUN) setMTU(mtu int) error {
	return netsh("interface", "ipv4", "set", "subinterface", t.Iface.Name(), "mtu=", fmt.Sprintf("%d", mtu))
}

// setAddress sets the interface's destination address and subnet.
func (t *TUN) setAddress(address string) error {
	return netsh("interface", "ip", "set", "address", "name=", t.Iface.Name(), "static", address)
}

// SetDestAddress isn't supported under Windows.
// You should instead use set address to set the interface to handle
// all addresses within a subnet.
func (t *TUN) setDestAddress(address string) error {
	return errors.New("destination addresses are not supported under windows")
}

// Up brings up an interface to allow it to start accepting connections.
func (t *TUN) Up() error {
	return nil
}

// Down brings down an interface stopping active connections.
func (t *TUN) Down() error {
	return nil
}

// Delete removes a TUN device from the host.
func Delete(name string) error {
	return nil
}

func netsh(args ...string) (err error) {
	cmd := exec.Command("netsh", args...)
	err = cmd.Run()
	return
}
