//go:build unix

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"os"
)

// ioctl performs a system-level I/O control operation.
func ioctl(fd uintptr, request uintptr, argv uintptr) error {
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, request, argv); errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

// GetAdapterName returns the current name of the Unix interface.
func (a *SwiftInterface) GetAdapterName() (string, error) {
	if a.name == "" {
		return "", errors.New("adapter name is not set")
	}
	return a.name, nil
}

// GetAdapterIndex retrieves the system-wide link index of the interface.
func (a *SwiftInterface) GetAdapterIndex() (int, error) {
	if a.index != 0 {
		return a.index, nil
	}
	return 0, errors.New("adapter index not initialized")
}

// SetMTU updates the Maximum Transmission Unit for the Unix interface.
func (a *SwiftInterface) SetMTU(mtu int) error {
	name, err := a.GetAdapterName()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	if err = netlink.LinkSetMTU(link, mtu); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

// SetUnicastIpAddressEntry assigns an IP address and optional gateway to the interface.
func (a *SwiftInterface) SetUnicastIpAddressEntry(config *swiftypes.UnicastConfig) error {
	name, err := a.GetAdapterName()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface: %w", err)
	}

	if err := netlink.AddrAdd(link, &netlink.Addr{
		IPNet: config.IPNet,
	}); err != nil {
		return fmt.Errorf("failed to add address %v to interface %s: %v", config.IPNet, name, err)
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

// SetStatus toggles the interface between Up and Down states.
func (a *SwiftInterface) SetStatus(status swiftypes.InterfaceStatus) error {
	name, err := a.GetAdapterName()
	if err != nil {
		return err
	}

	link, err := netlink.LinkByName(name)
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

// AddRoute adds a network route via the current interface.
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

// RemoveRoute remove a network route via the current interface.
func (a *SwiftInterface) RemoveRoute(route *netlink.Route) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	route.LinkIndex = index

	if err := netlink.RouteDel(route); err != nil {
		return fmt.Errorf("failed to add route %v: %v", route, err)
	}

	return nil
}

// ReplaceRoute replace a network route via the current interface.
func (a *SwiftInterface) ReplaceRoute(route *netlink.Route) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	route.LinkIndex = index

	if err := netlink.RouteReplace(route); err != nil {
		return fmt.Errorf("failed to add route %v: %v", route, err)
	}

	return nil
}

// ChangeRoute change a network route via the current interface.
func (a *SwiftInterface) ChangeRoute(route *netlink.Route) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	route.LinkIndex = index

	if err := netlink.RouteChange(route); err != nil {
		return fmt.Errorf("failed to add route %v: %v", route, err)
	}

	return nil
}

// AppendRoute append a network route via the current interface.
func (a *SwiftInterface) AppendRoute(route *netlink.Route) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	route.LinkIndex = index

	if err := netlink.RouteAppend(route); err != nil {
		return fmt.Errorf("failed to add route %v: %v", route, err)
	}

	return nil
}

// RouteList remove a network route via the current interface.
func (a *SwiftInterface) RouteList(family int) ([]netlink.Route, error) {
	name, err := a.GetAdapterName()
	if err != nil {
		return nil, err
	}

	byName, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed reterive routes %s: %v", name, err)
	}

	return netlink.RouteList(byName, family)
}

// SetDNS is currently unsupported on Unix-like SwiftInterfaces.
func (a *SwiftInterface) SetDNS(_ *swiftypes.DNSConfig) error {
	return errors.New("DNS configuration not supported on this platform")
}
