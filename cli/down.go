package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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

// DownRun handles the execution of the down command.
func DownRun(r *cmd.Root, c *cmd.Sub) {
	// Parse Command Args
	args := c.Args.(*DownArgs)

	// Parse Global Config Flag for Custom Config Path
	configPath := r.Flags.(*GlobalFlags).Config
	if configPath == "" {
		configPath = "/etc/hyprspace/" + args.InterfaceName + ".yaml"
	}

	// Read lock from file system to stop process.
	lockPath := filepath.Join(filepath.Dir(configPath), args.InterfaceName+".lock")
	out, err := os.ReadFile(lockPath)
	checkErr(err)

	pid, err := strconv.Atoi(string(out))
	checkErr(err)

	process, err := os.FindProcess(pid)
	checkErr(err)

	err0 := process.Signal(os.Interrupt)

	err1 := tun.Delete(args.InterfaceName)

	// Different types of systems may need the tun devices destroyed first or
	// the process to exit first don't worry as long as one of these two has
	// suceeded.
	if err0 != nil && err1 != nil {
		checkErr(err0)
		checkErr(err1)
	}

	fmt.Println("[+] deleted hyprspace " + args.InterfaceName + " daemon")
}
