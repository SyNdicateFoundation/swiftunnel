//go:build !windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"net"
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

	addr := &netlink.Addr{IPNet: &net.IPNet{IP: config.IP, Mask: config.IPNet.Mask}}
	if err := netlink.AddrAdd(link, addr); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return nil
		}
		return fmt.Errorf("failed to add IP address: %w", err)
	}

	return nil
}

func setDNS(dnsServers []string) error {
	return errors.New("DNS configuration not supported on this platform")
}
