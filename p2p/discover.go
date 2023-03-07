package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/hyprspace/hyprspace/config"
	"github.com/hyprspace/hyprspace/state"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Discover starts up a DHT based discovery system finding and adding nodes with the same rendezvous string.
func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, peerTable map[string]peer.ID, i string) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	verbose := ctx.Value(config.WithVerbose) != nil
	if verbose {
		fmt.Println("[+] Starting Discover thread")
	}

	s := make(state.ConnectionState)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for ip, id := range peerTable {
				s[ip] = h.Network().Connectedness(id) == network.Connected
				if !s[ip] {
					addrs, err := dht.FindPeer(ctx, id)
					if err != nil {
						if verbose {
							fmt.Printf("[!] Couldn't find Peer(%s): %v\n", id, err)
						}
						continue
					}
					_, err = h.Network().DialPeer(ctx, addrs.ID)
					if err != nil {
						if verbose {
							fmt.Printf("[!] Couldn't dial Peer(%s): %v\n", id, err)
						}
						continue
					}
				}

				if verbose {
					fmt.Printf("[+] Connection to %s is alive\n", ip)
				}
			}

			state.Save(i, s)
		}
	}
}
