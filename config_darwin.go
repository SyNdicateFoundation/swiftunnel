package swiftunnel

import (
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"net"
)

type DriverType int

const (
	DriverTypeTunTapOSX DriverType = iota
	DriverTypeSystem
)

type Config struct {
	AdapterName string
	AdapterType swiftypes.AdapterType
	DriverType  DriverType

	MTU           int
	UnicastConfig *swiftypes.UnicastConfig
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
	}
}
