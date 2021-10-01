package cli

import (
	"fmt"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/daemon"
)

// Peers creates and brings up a Hyprspace Interface.
var Peers = cmd.Sub{
	Name:  "peers",
	Alias: "p",
	Short: "List peers currently connected to an interface.",
	Args:  &PeersArgs{},
	Run:   PeersRun,
}

// PeersArgs handles the specific arguments for the peers command.
type PeersArgs struct {
	InterfaceName string
}

// PeersRun handles the execution of the up command.
func PeersRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*PeersArgs)

	peers, err := daemon.GetConnectedPeers(args.InterfaceName)

	if err != nil && err.Error() != "" {
		fmt.Println("Failed to get peers:", err)
	} else {
		fmt.Printf("Peers connected on %s: %d\n", args.InterfaceName, len(peers))
		for _, p := range peers {
			fmt.Println(p)
		}
	}
}
