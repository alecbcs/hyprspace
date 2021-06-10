package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/sethvargo/go-diceware/diceware"
	"gopkg.in/yaml.v2"
)

// Init creates a configuration for a Hyprspace Interface.
var Init = cmd.Sub{
	Name:  "init",
	Alias: "i",
	Short: "Initialize An Interface Config",
	Args:  &InitArgs{},
	Run:   InitRun,
}

// InitArgs handles the specific arguments for the init command.
type InitArgs struct {
	InterfaceName string
}

// InitRun handles the execution of the init command.
func InitRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Arguments
	args := c.Args.(*InitArgs)

	// Parse Global Config Flag
	configPath := r.Flags.(*GlobalFlags).Config
	if configPath == "" {
		configPath = "/etc/hyprspace/" + args.InterfaceName + ".yaml"
	}

	// Create New Libp2p Node
	host, err := libp2p.New(context.Background())
	checkErr(err)

	// Get Node's Private Key
	keyBytes, err := crypto.MarshalPrivateKey(host.Peerstore().PrivKey(host.ID()))
	checkErr(err)

	// Generate a random diceware discovery key
	list, err := diceware.Generate(4)
	checkErr(err)

	// Setup an initial default command.
	new := config.Config{
		Interface: config.Interface{
			Name:        args.InterfaceName,
			Address:     "10.1.1.1/24",
			ID:          host.ID().Pretty(),
			PrivateKey:  string(keyBytes),
			DiscoverKey: strings.Join(list, "-"),
		},
	}

	out, err := yaml.Marshal(&new)
	checkErr(err)

	err = os.MkdirAll(filepath.Dir(configPath), os.ModePerm)
	checkErr(err)

	f, err := os.Create(configPath)
	checkErr(err)

	// Write out config to file.
	_, err = f.Write(out)
	checkErr(err)

	f.Close()
}
