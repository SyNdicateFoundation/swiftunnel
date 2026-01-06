//go:build unix

package gateway

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
)

// DiscoverGatewayIPv4 finds the IPv4 default gateway on Unix-like systems.
func DiscoverGatewayIPv4() (net.IP, error) {
	return discoverGateway(netlink.FAMILY_V4)
}

// DiscoverGatewayIPv6 finds the IPv6 default gateway on Unix-like systems.
func DiscoverGatewayIPv6() (net.IP, error) {
	return discoverGateway(netlink.FAMILY_V6)
}

// discoverGateway iterates through the routing table to find the gateway for the specified family.
func discoverGateway(family int) (net.IP, error) {
	routes, err := netlink.RouteList(nil, family)
	if err != nil {
		return nil, fmt.Errorf("failed to get route list: %w", err)
	}

	for _, route := range routes {
		if (route.Dst == nil || route.Dst.IP.IsUnspecified()) && route.Gw != nil {
			return route.Gw, nil
		}
	}
	return nil, ErrNoGateway
}
