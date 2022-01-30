//go:build windows
// +build windows

package tun

import (
	"errors"
	"fmt"
	"net"
	"os/exec"

	"github.com/songgao/water"
)

// New creates and returns a new TUN interface for the application.
func New(name string, opts ...Option) (*TUN, error) {
	result := TUN{}

	// Apply options early to set struct values for interface creation.
	err := result.Apply(opts...)
	if err != nil {
		return nil, err
	}

	// TUN on Windows requires address and network to be set on device creation stage
	// We also set network to 0.0.0.0/0 so we able to reach networks behind the node
	// https://github.com/songgao/water/blob/master/params_windows.go
	// https://gitlab.com/openconnect/openconnect/-/blob/master/tun-win32.c
	ip, _, err := net.ParseCIDR(result.Src)
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

	// Interface should be enabled before creation of water interface
	// Otherwise there will be an error "The system cannot find the file specified."
	netsh("interface", "set", "interface", "name=", name, "enable")

	// Create Water Interface
	iface, err := water.New(cfg)
	if err != nil {
		return nil, err
	}

	// Set TUN interface to newly created interface
	result.Iface = iface

	// Apply options to setup TUN interface configuration
	// Setup interface address
	err = result.setupAddress(result.Src)
	if err != nil {
		return nil, err
	}

	// Setup interface mtu size
	err = result.setupMTU(result.MTU)
	if err != nil {
		return nil, err
	}

	return &result, err
}

// setMTU configures the interface's MTU.
func (t *TUN) setMTU(mtu int) error {
	t.MTU = mtu
	return nil
}

// setAddress configures the interface's address.
func (t *TUN) setAddress(address string) error {
	t.Src = address
	return nil
}

// setupMTU sets the Maximum Tansmission Unit Size for a
// Packet on the interface.
func (t *TUN) setupMTU(mtu int) error {
	return netsh("interface", "ipv4", "set", "subinterface", t.Iface.Name(), "mtu=", fmt.Sprintf("%d", mtu))
}

// setupAddress sets the interface's destination address and subnet.
func (t *TUN) setupAddress(address string) error {
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
	return netsh("interface", "set", "interface", "name=", name, "disable")
}

func netsh(args ...string) (err error) {
	cmd := exec.Command("netsh", args...)
	err = cmd.Run()
	return
}
