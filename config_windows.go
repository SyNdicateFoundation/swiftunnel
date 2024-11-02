//go:build windows

package swiftunnel

import (
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"net"
)

type DriverType int

const (
	DriverTypeWintun DriverType = iota
	DriverTypeOpenVPN
)

type Config struct {
	AdapterName     string
	AdapterTypeName string
	AdapterGUID     swiftypes.GUID
	AdapterType     swiftypes.AdapterType
	DriverType      DriverType

	MTU           int
	DNSConfig     *swiftypes.DNSConfig
	UnicastConfig *swiftypes.UnicastConfig
}

func NewDefaultConfig() Config {
	ip, ipNet, err := net.ParseCIDR("10.18.21.1/24")
	if err != nil {
		panic(err)
	}

	return Config{
		AdapterName:     "Swiftunnel VPN",
		AdapterTypeName: "Swiftunnel",
		AdapterType:     swiftypes.AdapterTypeTUN,
		DriverType:      DriverTypeWintun,
		MTU:             1500,
		DNSConfig: &swiftypes.DNSConfig{
			DnsServers: []net.IP{
				net.IPv4(8, 8, 8, 8),
				net.IPv4(8, 8, 4, 4),
			},
		},
		UnicastConfig: &swiftypes.UnicastConfig{
			IP:       ip,
			IPNet:    ipNet,
			DadState: swiftypes.IpDadStatePreferred,
		},
	}
}
