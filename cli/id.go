package cli

import (
	"fmt"
	"strings"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
)

var Id = cmd.Sub{
	Name:  "id",
	Alias: "this",
	Short: "Print this Node's Peer ID for given Interface",
	Args:  &UpArgs{},
	Flags: &IdFlags{},
	Run:   IdRun,
}

type IdFlags struct {
	Yaml bool `long:"yaml" desc:"Print info as an Interface YAML config chunk"`
	Cmd  bool `long:"cmd" desc:"Print info as add command arguments"`
}

func IdRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*UpArgs)

	// Parse Command Flags
	flags := c.Flags.(*IdFlags)

	// Parse Global Config Flag for Custom Config Path
	configPath := r.Flags.(*GlobalFlags).Config
	if configPath == "" {
		configPath = "/etc/hyprspace/" + args.InterfaceName + ".yaml"
	}

	// Read in configuration from file.
	cfg, err := config.Read(configPath)
	checkErr(err)

	addr := strings.Split(cfg.Interface.Address, "/")[0]

	if flags.Yaml {
		fmt.Printf("  %s:\n    id: %s\n", addr, cfg.Interface.ID)
	} else if flags.Cmd {
		fmt.Printf("%s %s %s\n", cfg.Interface.Name, addr, cfg.Interface.ID)
	} else {
		fmt.Printf("Peer ID: %s\nAddress: %s\n", cfg.Interface.ID, addr)
	}
}
