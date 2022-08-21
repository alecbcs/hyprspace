package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

// Discover starts up a DHT based discovery system finding and adding nodes with the same rendezvous string.
func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, ip string, id peer.ID) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if h.Network().Connectedness(id) != network.Connected {
				addr, err := dht.FindPeer(ctx, id)
				if err != nil {
					continue
				}
				id = addr.ID

				stream, err := h.NewStream(ctx, id, Protocol)
				if err != nil {
					continue
				}
				fmt.Printf("[+] Connection to %s Successful. Network Ready.\n", ip)
				_ = stream.Close()
			}
		}
	}
}
