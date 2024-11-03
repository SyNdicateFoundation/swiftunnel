//go:build !windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"github.com/vishvananda/netlink"
)

func setMTU(ifaceName string, mtu int) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	err = netlink.LinkSetMTU(link, mtu)
	if err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

func setUnicastIpAddressEntry(ifaceName string, config *swiftypes.UnicastConfig) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	// Add the IP address to the interface
	if err := netlink.AddrAdd(link, &netlink.Addr{
		IPNet: config.IPNet,
	}); err != nil {
		return fmt.Errorf("failed to add address %v to interface %s: %v", config.IPNet, ifaceName, err)
	}

	// If a gateway is specified, add a route for it
	if config.Gateway != nil {
		if err := netlink.RouteAdd(&netlink.Route{
			LinkIndex: link.Attrs().Index,
			Gw:        config.Gateway,
		}); err != nil {
			return fmt.Errorf("failed to add route to gateway %v: %v", config.Gateway, err)
		}
	}

	return nil
}

func setDNS(dnsServers []string) error {
	return errors.New("DNS configuration not supported on this platform")
}
