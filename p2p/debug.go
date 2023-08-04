package p2p

import (
	"fmt"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func DebugEvents(node host.Host, dht *dht.IpfsDHT) {
	fmt.Println("Available Event Types")
	for _, et := range node.EventBus().GetAllEventTypes() {
		fmt.Printf("- %s\n", et)
	}

	sub, err := node.EventBus().Subscribe(event.WildcardSubscription)
	checkErr(err)
	for e := range sub.Out() {
		switch e := e.(type) {
		case event.EvtLocalProtocolsUpdated:
			fmt.Printf("Local Protocols Updated\nAdded: %v\nRemoved: %v", e.Added, e.Removed)
		case event.EvtLocalAddressesUpdated:
			fmt.Println("Local Addresses Updated\nCurrent:")
			for _, addr := range e.Current {
				fmt.Printf("- %s | ", addr.Address.String())
				if addr.Action == event.Added {
					fmt.Println("Added")
				} else if addr.Action == event.Maintained {
					fmt.Println("Maintained")
				} else {
					fmt.Println("Removed or Unknown")
				}
			}
			if len(e.Removed) > 0 {
				fmt.Println("Removed:")
				for _, addr := range e.Removed {
					fmt.Printf("- %s\n", addr.Address.String())
				}
			}

		case event.EvtPeerConnectednessChanged:
			continue
		case event.EvtLocalReachabilityChanged:
			fmt.Printf("Local Reachability Changed: %s\n", e.Reachability.String())
		case event.EvtNATDeviceTypeChanged:
			fmt.Printf("NAT DeviceType Changed: %s - %s\n", e.TransportProtocol.String(), e.NatDeviceType.String())
		default:
			// fmt.Println("Unhandled", e)
			continue
		}
	}
}
