package tun

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

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

func SetMTU(name string, mtu int) (err error) {
	return ip("link", "set", "dev", name, "mtu", fmt.Sprintf("%d", mtu))
}

func SetAddress(name string, address string) (err error) {
	return ip("addr", "add", address, "dev", name)
}

func Up(name string) (err error) {
	return ip("link", "set", "dev", name, "up")
}

func Down(name string) (err error) {
	return ip("link", "set", "dev", name, "down")
}

func Delete(name string) (err error) {
	return ip("link", "delete", "dev", name)
}

func ip(args ...string) (err error) {
	cmd := exec.Command("/sbin/ip", args...)
	err = cmd.Run()
	return
}
