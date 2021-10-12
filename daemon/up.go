package daemon

import (
	"github.com/hyprspace/hyprspace/p2p"
)

type UpReq struct {
	Interface  string
	ConfigPath string
}

type UpResp struct {
	Error string
}

type UpHandler struct {
	D *Daemon
}

func (h *UpHandler) Execute(req UpReq, res *UpResp) (err error) {
	if _, ok := h.D.Interfaces[req.Interface]; ok {
		res.Error = "Interface already up"
		return
	}

	i, err1 := p2p.Up(req.Interface, req.ConfigPath)
	if err1 != nil {
		res.Error = err1.Error()
	}
	h.D.Interfaces[req.Interface] = i
	return nil
}
