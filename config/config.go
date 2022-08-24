package config

import (
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v2"
)

// Config is the main Configuration Struct for Hyprspace.
type Config struct {
	Path      string           `yaml:"path,omitempty"`
	Interface Interface        `yaml:"interface"`
	Peers     map[string]Peer  `yaml:"peers"`
	Routes    map[string]Route `yaml:"routes"`
}

// Interface defines all of the fields that a local node needs to know about itself!
type Interface struct {
	Name       string `yaml:"name"`
	ID         string `yaml:"id"`
	ListenPort int    `yaml:"listen_port"`
	Address    string `yaml:"address"`
	PrivateKey string `yaml:"private_key"`
}

// Peer defines a peer in the configuration. We might add more to this later.
type Peer struct {
	ID string `yaml:"id"`
}

type Route struct {
	IP string `yaml:"ip"`
}

// Read initializes a config from a file.
func Read(path string) (*Config, error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := Config{
		Interface: Interface{
			Name:       "hs0",
			ListenPort: 8001,
			Address:    "10.1.1.1/24",
			ID:         "",
			PrivateKey: "",
		},
	}

	// Read in config settings from file.
	err = yaml.Unmarshal(in, &result)
	if err != nil {
		return nil, err
	}

	// Check peers have valid ip addresses
	for ip := range result.Peers {
		if net.ParseIP(ip).String() == "<nil>" {
			return nil, fmt.Errorf("%s is not a valid ip address", ip)
		} else {
			fmt.Printf("[+] Assign this ip: %s to node: %s.\n", ip, result.Peers[ip].ID)
		}
	}

	for route := range result.Routes {
		_, _, err := net.ParseCIDR(route)
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid route", route)
		} else {
			fmt.Printf("[+] Assign route %s via %s.\n", route, result.Routes[route].IP)
		}
	}

	// Overwrite path of config to input.
	result.Path = path
	return &result, nil
}
