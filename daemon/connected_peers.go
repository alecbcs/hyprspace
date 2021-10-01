package daemon

import (
	"github.com/hyprspace/hyprspace/p2p"
)

type PeersReq struct {
	Interface string `json:"Interface"`
}

type PeersResp struct {
	ConnectedPeers []string
	Error          string
}

type PeersHandler struct {
	D *Daemon
}

func (h *PeersHandler) Execute(req PeersReq, res *PeersResp) (err error) {
	hyprspace, ok := h.D.Interfaces[req.Interface]
	if !ok {
		res.Error = "Interface is not running"
		return
	}

	for ip, np := range hyprspace.PeerTable {
		if p2p.PeerHasStreams(hyprspace.Node, np.PeerID) {
			res.ConnectedPeers = append(res.ConnectedPeers, ip)
		}
	}

	return
}
