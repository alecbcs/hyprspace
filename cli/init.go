package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
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
	host, err := libp2p.New()
	checkErr(err)

	// Get Node's Private Key
	keyBytes, err := crypto.MarshalPrivateKey(host.Peerstore().PrivKey(host.ID()))
	checkErr(err)

	// Setup an initial default command.
	new := config.Config{
		Interface: config.Interface{
			Name:       args.InterfaceName,
			ListenPort: 8001,
			Address:    "10.1.1.1/24",
			ID:         host.ID().Pretty(),
			PrivateKey: string(keyBytes),
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

	err = f.Close()
	checkErr(err)

	// Print config creation message to user
	fmt.Printf("Initialized new config at %s\n", configPath)
	fmt.Println("To edit the config run,")
	fmt.Println()
	if strings.HasPrefix(configPath, "/etc/") {
		fmt.Printf("    sudo nano %s\n", configPath)
	} else {
		fmt.Printf("    nano %s\n", configPath)
	}
	fmt.Println()
}
