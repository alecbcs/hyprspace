package cli

import (
	"fmt"
	"os"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/hyprspace/hyprspace/daemon"
)

// Daemon brings the daemon up or down
var Daemon = cmd.Sub{
	Name:  "daemon",
	Alias: "b",
	Short: "Control the daemon.",
	Args:  &DaemonArgs{},
	Run:   DaemonRun,
}

type DaemonArgs struct {
	UpDown string
}

// Brings the daemon up or down
func DaemonRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*DaemonArgs)

	if args.UpDown == "up" {
		fmt.Println("Starting hyprspace daemon on port", daemon.PORT)
		out := make(chan error)
		go daemon.Run(out)

		select {
		case err := <-out:
			if err == nil {
				fmt.Println("Daemon shutting down")
				os.Exit(0)
			}
			fmt.Println(err)
			os.Exit(1)
		}
		// At this point the daemon has shut down
	} else if args.UpDown == "down" {
		daemon.Shutdown()
	}
}
