//go:build linux
// +build linux

package tun

import (
	"errors"

	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
)

// New creates and returns a new TUN interface for the application.
func New(name string, address string, opts ...Option) (*TUN, error) {
	// Setup TUN Config
	cfg := water.Config{
		DeviceType: water.TUN,
	}
	cfg.Name = name

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
	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.LinkSetMTU(link, mtu)
}

// setDestAddress sets the interface's destination address and subnet.
func (t *TUN) setAddress(address string) error {
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.AddrAdd(link, addr)
}

// SetDestAddress isn't supported under Linux.
// You should instead use set address to set the interface to handle
// all addresses within a subnet.
func (t *TUN) setDestAddress(address string) error {
	return errors.New("destination addresses are not supported under linux")
}

// Up brings up an interface to allow it to start accepting connections.
func (t *TUN) Up() error {
	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(link)
}

// Down brings down an interface stopping active connections.
func (t *TUN) Down() error {
	link, err := netlink.LinkByName(t.Iface.Name())
	if err != nil {
		return err
	}
	return netlink.LinkSetDown(link)
}

// Delete removes a TUN device from the host.
func Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkDel(link)
}
