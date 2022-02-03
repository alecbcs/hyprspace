package tun

import "github.com/songgao/water"

// TUN is a struct containing the fields necessary
// to configure a system TUN device. Access the
// internal TUN device through TUN.Iface
type TUN struct {
	Iface *water.Interface
	MTU   int
	Src   string
	Dst   string
}

// Apply configures the specified options for a TUN device.
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
