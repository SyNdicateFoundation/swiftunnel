//go:build !windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"github.com/vishvananda/netlink"
	"os"
	"syscall"
)

func ioctl(fd uintptr, request uintptr, argv uintptr) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, argv); errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

func (a *SwiftInterface) GetAdapterName() (string, error) {
	if a.name == "" {
		return "", errors.New("adapter name is not set")
	}
	return a.name, nil
}

func (a *SwiftInterface) GetAdapterIndex() (int, error) {
	if a.name == "" {
		return 0, errors.New("adapter name is not set")
	}

	ifi, err := netlink.LinkByName(a.name)
	if err != nil {
		return 0, err
	}

	return ifi.Attrs().Index, nil
}

func (a *SwiftInterface) SetMTU(mtu int) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByIndex(index)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	if err = netlink.LinkSetMTU(link, mtu); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

func (a *SwiftInterface) SetUnicastIpAddressEntry(config *swiftypes.UnicastConfig) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByIndex(index)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	if err := netlink.AddrAdd(link, &netlink.Addr{
		IPNet: config.IPNet,
	}); err != nil {
		return fmt.Errorf("failed to add address %v to interface %d: %v", config.IPNet, index, err)
	}

	if config.Gateway != nil {
		if err := a.AddRoute(&netlink.Route{
			LinkIndex: link.Attrs().Index,
			Gw:        config.Gateway,
		}); err != nil {
			return fmt.Errorf("failed to add route to gateway %v: %v", config.Gateway, err)
		}
	}

	return nil
}

func (a *SwiftInterface) SetStatus(status swiftypes.InterfaceStatus) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByIndex(index)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	switch status {
	case swiftypes.InterfaceUp:
		return netlink.LinkSetUp(link)
	case swiftypes.InterfaceDown:
		return netlink.LinkSetDown(link)
	}

	return nil
}

func (a *SwiftInterface) AddRoute(route *netlink.Route) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	route.LinkIndex = index

	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add route %v: %v", route, err)
	}

	return nil
}

func (a *SwiftInterface) SetDNS(config *swiftypes.DNSConfig) error {
	return errors.New("DNS configuration not supported on this platform")
}
