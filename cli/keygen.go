package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDrake/cli-ng/v2/cmd"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/sethvargo/go-diceware/diceware"
)

// KeyGen saves a node authentication for a Hyprspace Interface to files.
var KeyGen = cmd.Sub{
	Name:	"keygen",
	Alias:	"k",
	Short:	"Generate authentication keys",
	Flags:	&KeyGenFlags{},
	Run:	KeyGenRun,
}

// KeyGenFlags handles the specific flags for the keygen command.
type KeyGenFlags struct {
	OutPath string `short:"k" long:"keygen" desc:"Generate authentication keys"`
}

// Create New Libp2p Node
func CreateNode() host.Host {
	host, err := libp2p.New(context.Background())
	checkErr(err)
	return host
}

// Get a Node's Private Key as a String
func GetPrivateKey(host host.Host) string {
	keyBytes, err := crypto.MarshalPrivateKey(host.Peerstore().PrivKey(host.ID()))
	checkErr(err)
	return string(keyBytes)
}

// Get a Node's ID as a String
func GetID(host host.Host) string {
	return host.ID().Pretty()
}

// Generate a random diceware discovery key
func GenerateDiscoveryKey() string {
	list, err := diceware.Generate(4)
	checkErr(err)
	return strings.Join(list, "-")
}

// Write Generated Keys to File
func WriteFile(outPath string, fileName string, content string) {
	path := fmt.Sprintf("%s/%s", outPath, fileName)
    err := os.WriteFile(path, []byte(content), 0077)
    checkErr(err)
}

// KeyGenRun handles the execution of the node command.
func KeyGenRun(r *cmd.Root, c *cmd.Sub) {

	outPath := c.Flags.(*KeyGenFlags).OutPath
	if outPath == "" {
		outPath = "~/.hyprspace"
	}

	host := CreateNode()
	PrivateKey := GetPrivateKey(host)
	ID := GetID(host)
	DiscoverKey := GenerateDiscoveryKey()

	err := os.MkdirAll(filepath.Dir(outPath), os.ModePerm)
	checkErr(err)

	WriteFile(outPath, "key.hyprspace", PrivateKey)
	WriteFile(outPath, "id.hyprspace", ID)
	WriteFile(outPath, "discover.hyprspace", DiscoverKey)
}
