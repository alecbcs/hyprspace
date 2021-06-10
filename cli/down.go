package cli

import (
	"fmt"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/tun"
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

func DownRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*DownArgs)

	fmt.Println("[+] ip link delete dev " + args.InterfaceName)
	err := tun.Delete(args.InterfaceName)
	checkErr(err)
}
