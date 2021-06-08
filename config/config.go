package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Interface Interface       `yaml:"interface"`
	Peers     map[string]Peer `yaml:"peers"`
}

type Interface struct {
	Name        string `yaml:"name"`
	Address     string `yaml:"address"`
	ID          string `yaml:"id"`
	DiscoverKey string `yaml:"discover_key"`
	PrivateKey  string `yaml:"private_key"`
}

type Peer struct {
	ID string `yaml:"id"`
}

// Read initalizes a config from a file.
func Read(path string) (result Config, err error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(in, &result)
	return
}
