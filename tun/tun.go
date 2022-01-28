package tun

import "github.com/songgao/water"

type TUN struct {
	Iface *water.Interface
	Src   string
	Dst   string
}

func (t *TUN) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(t); err != nil {
			return err
		}
	}
	return nil
}
