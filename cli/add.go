package cli

import (
	"fmt"
	"net"
	"os"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/libp2p/go-libp2p/core/peer"
	"gopkg.in/yaml.v2"
)

var Add = cmd.Sub{
	Name:  "add",
	Alias: "add",
	Short: "Add Peer to Hyprspace Interface",
	Args:  &AddArgs{},
	Flags: &AddFlags{},
	Run:   AddRun,
}

type AddArgs struct {
	InterfaceName string
	Address       string
	ID            string
}

type AddFlags struct {
	Overwrite bool `long:"overwrite" desc:"Overwrite existing Address"`
}

func AddRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*AddArgs)

	// Parse Command Flags
	flags := c.Flags.(*AddFlags)

	if net.ParseIP(args.Address).String() == "<nil>" {
		fmt.Printf("%s is not a valid ip address\n", args.Address)
		return
	}

	// Parse Global Config Flag for Custom Config Path
	configPath := r.Flags.(*GlobalFlags).Config
	if configPath == "" {
		configPath = "/etc/hyprspace/" + args.InterfaceName + ".yaml"
	}

	// Read in configuration from file.
	cfg, err := config.Read(configPath)
	checkErr(err)

	if id, exists := cfg.Peers[args.Address]; exists && !flags.Overwrite {
		fmt.Printf("Address %s is already used by Peer(%s).\nUse --overwrite flag if you'd like to do so.\n", args.Address, id.ID)
		return
	}

	_, err = peer.Decode(args.ID)
	if err != nil {
		fmt.Printf("%s is not a valid Peer ID: %v\n", args.ID, err)
		return
	}

	cfg.Peers[args.Address] = config.Peer{ID: args.ID}
	out, err := yaml.Marshal(cfg)
	checkErr(err)

	// Backup current config
	err = os.Rename(configPath, configPath+".backup")
	checkErr(err)

	f, err := os.Create(configPath)
	checkErr(err)

	// Write out config to file.
	_, err = f.Write(out)
	checkErr(err)

	fmt.Printf("Successfuly added new Peer to the config at %s\n", configPath)
	fmt.Println("To Apply changes restart Hyprspace daemon")
}
