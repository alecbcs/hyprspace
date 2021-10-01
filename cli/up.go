package cli

import (
	"fmt"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/config"
	"github.com/hyprspace/hyprspace/daemon"
	"github.com/songgao/water"
)

var (
	// Global is the global interface configuration for the
	// application instance.
	Global config.Config
	// iface is the tun device used to pass packets between
	// Hyprspace and the user's machine.
	iface *water.Interface
)

// Up creates and brings up a Hyprspace Interface.
var Up = cmd.Sub{
	Name:  "up",
	Alias: "up",
	Short: "Create and Bring Up a Hyprspace Interface.",
	Args:  &UpArgs{},
	Flags: &UpFlags{},
	Run:   UpRun,
}

// UpArgs handles the specific arguments for the up command.
type UpArgs struct {
	InterfaceName string
}

// UpFlags handles the specific flags for the up command.
type UpFlags struct {
	Foreground bool `short:"f" long:"foreground" desc:"Don't Create Background Daemon."`
}

// UpRun handles the execution of the up command.
func UpRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*UpArgs)

	// Parse Command Flags
	//flags := c.Flags.(*UpFlags)

	// Parse Global Config Flag for Custom Config Path
	configPath := r.Flags.(*GlobalFlags).Config
	if configPath == "" {
		configPath = "/etc/hyprspace/" + args.InterfaceName + ".yaml"
	}

	err := daemon.UpInterface(args.InterfaceName, configPath)

	if err != nil && err.Error() != "" {
		fmt.Println("Failed to start interface:", err)
	} else {
		fmt.Println("[+] Started hyprspace interface", args.InterfaceName)
	}
}
