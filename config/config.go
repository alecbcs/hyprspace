package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config is the main Configuration Struct for Hyprspace.
type Config struct {
	Interface Interface       `yaml:"interface"`
	Peers     map[string]Peer `yaml:"peers"`
}

// Interface defines all of the fields that a local node needs to know about itself!
type Interface struct {
	Name        string `yaml:"name"`
	Address     string `yaml:"address"`
	ID          string `yaml:"id"`
	DiscoverKey string `yaml:"discover_key"`
	PrivateKey  string `yaml:"private_key"`
}

// Peer defines a peer in the configuration. We might add more to this later.
type Peer struct {
	ID string `yaml:"id"`
}

// Read initializes a config from a file.
func Read(path string) (result Config, err error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(in, &result)
	return
}
