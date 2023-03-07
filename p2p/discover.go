package p2p

import (
	"context"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Discover starts up a DHT based discovery system finding and adding nodes with the same rendezvous string.
func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, peerTable map[string]peer.ID) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, id := range peerTable {
				if h.Network().Connectedness(id) != network.Connected {
					addrs, err := dht.FindPeer(ctx, id)
					if err != nil {
						continue
					}
					_, err = h.Network().DialPeer(ctx, addrs.ID)
					if err != nil {
						continue
					}
				}
			}
		}
	}
}
