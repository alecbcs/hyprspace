package daemon

import (
	"fmt"
	"os"
)

type ShutdownReq struct {
	Daemon    bool
	Interface string
}

type ShutdownResp struct {
	Error string
}

type ShutdownHandler struct {
	D *Daemon
}

func (h *ShutdownHandler) Execute(req ShutdownReq, res *ShutdownResp) (err error) {
	if req.Daemon {
		// Gracefully close all active interfaces and kill daemon
		fmt.Println("Shutting down daemon")
		for iface, h := range h.D.Interfaces {
			fmt.Println("Shutting down", iface)
			h.Shutdown()
		}

		os.Exit(0)
	}

	i, ok := h.D.Interfaces[req.Interface]
	if !ok {
		res.Error = "interface not up"
		return
	}
	errT := i.Shutdown()
	if errT != nil {
		res.Error = errT.Error()
	}
	delete(h.D.Interfaces, req.Interface)
	fmt.Println("Shut down interface", req.Interface)
	return nil
}
