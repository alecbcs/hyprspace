//go:build linux
// +build linux

package tun

import (
	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
)

// New creates and returns a new TUN interface for the application.
func New(name, address string) (result *water.Interface, err error) {
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
func SetMTU(name string, mtu int) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetMTU(link, mtu)
}

// SetAddress sets the interface's known address and subnet.
func SetAddress(name string, address string) error {
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.AddrAdd(link, addr)
}

// Up brings up an interface to allow it to start accepting connections.
func Up(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(link)
}

// Down brings down an interface stopping active connections.
func Down(name string) error {
	link, err := netlink.LinkByName(name)
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
