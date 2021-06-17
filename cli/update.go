package cli

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/inconshreveable/go-update"
	"github.com/tcnksm/go-latest"
)

// Update checks for a new version of the Hyprspace program and updates itself
// if a newer version is found and the user agrees to update.
var Update = cmd.Sub{
	Name:  "update",
	Alias: "upd",
	Short: "Update Hyprspace to the lastest version.",
	Args:  &UpdateArgs{},
	Flags: &UpdateFlags{},
	Run:   UpdateRun,
}

// UpdateArgs handles the specific arguments for the update command.
type UpdateArgs struct {
}

// UpdateFlags handles the specific flags for the update command.
type UpdateFlags struct {
	Yes bool `short:"y" long:"yes" desc:"If a newer version is found update without prompting the user."`
}

// UpdateRun handles the checking and self updating of the AIT program.
func UpdateRun(r *cmd.Root, c *cmd.Sub) {
	fmt.Printf("Current Version: %s\n", appVersion)

	flags := c.Flags.(*UpdateFlags)
	latestVersion := &latest.GithubTag{
		Owner:      "hyprspace",
		Repository: "hyprspace",
	}

	res, _ := latest.Check(latestVersion, appVersion)
	fmt.Printf("Latest Version: %s\n", res.Current)

	if res.Outdated {
		if !flags.Yes {
			fmt.Println("Would you like to update Hyprspace to the newest version? ([y]/n)")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.ToLower(strings.TrimSpace(input))
			if input == "n" {
				return
			}
		}
		url := "https://github.com/hyprspace/hyprspace/releases/download/v" + res.Current + "/hyprspace-v" + res.Current + "-" + runtime.GOOS + "-" + runtime.GOARCH

		doneChan := make(chan int, 1)
		wg := sync.WaitGroup{}
		wg.Add(1)

		// Display Spinner on Update.
		go SpinnerWait(doneChan, "Updating Hyprspace...", &wg)

		resp, err := http.Get(url)
		checkErr(err)

		defer resp.Body.Close()
		err = update.Apply(resp.Body, update.Options{})
		checkErr(err)

		doneChan <- 0
		wg.Wait()

		fmt.Print("\rUpdating Hyprspace: Done!\n")
	} else {
		fmt.Println("Already Up-To-Date!")
	}
}
