//go:build linux

package swiftunnel

import (
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"net"
)

type Permissions struct {
	Owner uint
	Group uint
}

type Config struct {
	AdapterName string
	AdapterType swiftypes.AdapterType

	MTU           int
	UnicastConfig *swiftypes.UnicastConfig

	MultiQueue  bool
	Permissions *Permissions
	Persist     bool
}

func NewPermissions(owner, group uint) *Permissions {
	return &Permissions{owner, group}
}

func NewDefaultConfig() Config {
	ip, ipNet, err := net.ParseCIDR("10.18.21.1/24")
	if err != nil {
		panic(err)
	}

	return Config{
		AdapterName: "Swiftunnel VPN",
		AdapterType: swiftypes.AdapterTypeTUN,
		MTU:         1500,
		UnicastConfig: &swiftypes.UnicastConfig{
			IPNet: ipNet,
			IP:    ip,
		},
		MultiQueue: false,
		Persist:    true,
	}
}
