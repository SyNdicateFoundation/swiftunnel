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

	MTU       int
	DNSConfig *swiftypes.DNSConfig
	UnicastIP *net.IPNet
}

func NewDefaultConfig() Config {
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
		UnicastIP: &net.IPNet{
			IP:   net.IPv4(10, 18, 21, 1),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
	}
}
