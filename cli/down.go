package cli

import (
	"fmt"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/daemon"
)

// Down brings down a Hyprspace interface and removes it from the system.
var Down = cmd.Sub{
	Name:  "down",
	Alias: "d",
	Short: "Bring Down A Hyprspace Interface.",
	Args:  &DownArgs{},
	Run:   DownRun,
}

// DownArgs handles the specific arguments for the down command.
type DownArgs struct {
	InterfaceName string
}

// DownRun handles the execution of the down command.
func DownRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*DownArgs)

	err := daemon.DownInterface(args.InterfaceName)
	if err != nil && err.Error() != "" {
		fmt.Println("Failed to bring down interface:", err)
	} else {
		fmt.Println("[-] Shutdown hyprspace interface", args.InterfaceName)
	}
}
