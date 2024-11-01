//go:build linux

package Swiftunnel

import (
	"github.com/XenonCommunity/Swiftunnel/swiftypes"
	"net"
)

type Permissions struct {
	Owner uint
	Group uint
}

type Config struct {
	AdapterName string
	AdapterType swiftypes.AdapterType

	MTU       int
	UnicastIP net.IPNet

	MultiQueue  bool
	Permissions *Permissions
	Persist     bool
}

func NewPermissions(owner, group uint) *Permissions {
	return &Permissions{owner, group}
}

func NewDefaultConfig() Config {
	return Config{
		AdapterName: "Swiftunnel VPN",
		AdapterType: swiftypes.AdapterTypeTUN,
		MTU:         1500,
		UnicastIP: net.IPNet{
			IP:   net.IPv4(10, 18, 21, 1),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
		Permissions: NewPermissions(0, 0),
		MultiQueue:  false,
		Persist:     true,
	}
}
