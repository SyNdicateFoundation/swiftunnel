//go:build windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/windows"
	"net"
	"strings"
	"unsafe"
)

var (
	iphlpapi                            = windows.NewLazySystemDLL("iphlpapi.dll")
	procCreateUnicastIpAddressEntry     = iphlpapi.NewProc("CreateUnicastIpAddressEntry")
	procInitializeUnicastIpAddressEntry = iphlpapi.NewProc("InitializeUnicastIpAddressEntry")
	procSetInterfaceDnsSettings         = iphlpapi.NewProc("SetInterfaceDnsSettings")
	procGetInterfaceDnsSettings         = iphlpapi.NewProc("GetInterfaceDnsSettings")
	procGetIfEntry                      = iphlpapi.NewProc("GetIfEntry")
	procSetIfEntry                      = iphlpapi.NewProc("SetIfEntry")
)

func (a *SwiftInterface) GetAdapterLUID() (swiftypes.LUID, error) {
	if a.service == nil {
		return swiftypes.NilLUID, ErrCannotFindAdapter
	}

	return a.service.GetAdapterLUID()
}

func (a *SwiftInterface) GetAdapterGUID() (swiftypes.GUID, error) {
	if a.service == nil {
		return swiftypes.NilGUID, ErrCannotFindAdapter
	}

	return a.service.GetAdapterGUID()
}

func (a *SwiftInterface) GetAdapterName() (string, error) {
	if a.service == nil {
		return "", ErrCannotFindAdapter
	}

	return a.service.GetAdapterName()
}

func (a *SwiftInterface) GetAdapterIndex() (int, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return a.service.GetAdapterIndex()
}

func (a *SwiftInterface) SetMTU(mtu int) error {
	adapterIndex, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	var ifRow windows.MibIfRow

	ifRow.Index = uint32(adapterIndex)

	ret, _, _ := procGetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to retrieve interface entry: %w", err)
	}

	ifRow.Mtu = uint32(mtu)
	ret, _, _ = procSetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

func (a *SwiftInterface) SetUnicastIpAddressEntry(config *swiftypes.UnicastConfig) error {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return err
	}

	var addressRow mibUnicastIPAddressRow

	// Initialize the addressRow structure
	_, _, _ = procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))

	if ipv4 := config.IP.To4(); ipv4 != nil {
		addressRow.Address.Family = windows.AF_INET
		copy(addressRow.Address.Addr[:net.IPv4len], ipv4)
	} else if ipv6 := config.IP.To16(); ipv6 != nil {
		addressRow.Address.Family = windows.AF_INET6
		copy(addressRow.Address.Addr[:net.IPv6len], ipv6)
	} else {
		return fmt.Errorf("invalid IP address: %s", config.IP)
	}

	// Get the prefix length from the mask
	ones, bits := config.IPNet.Mask.Size()
	if ones > bits {
		return fmt.Errorf("invalid subnet mask: %v", config.IPNet.Mask)
	}

	addressRow.InterfaceLUID = luid.ToUint64()
	addressRow.OnLinkPrefixLength = uint8(ones)
	addressRow.DadState = config.DadState

	ret, _, _ := procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) && !errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
		return fmt.Errorf("failed to create unicast IP address config: %w (error code: %d)", errno, ret)
	}

	return nil
}

func (a *SwiftInterface) SetDNS(config *swiftypes.DNSConfig) error {
	guid, err := a.GetAdapterGUID()
	if err != nil {
		return err
	}

	var settings dnsInterfaceSettings

	settings.Version = 1

	// Retrieve the current DNS interface settings
	ret, _, _ := procGetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&settings)),
	)
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to get DNS settings: %w", windows.Errno(ret))
	}

	// Set the DNS domain if provided
	if config.Domain != "" {
		domain, err := windows.UTF16PtrFromString(config.Domain)
		if err != nil {
			return fmt.Errorf("failed to convert domain to UTF16: %w", err)
		}
		settings.Domain = domain

		settings.Flags |= dnsSettingDomain
	}

	// Set the DNS servers if provided
	if len(config.DnsServers) > 0 {
		var servers []string
		var ipv6 bool

		for _, server := range config.DnsServers {
			if server.To4() == nil {
				ipv6 = true
			}

			servers = append(servers, server.String())
		}

		nameServer, err := windows.UTF16PtrFromString(strings.Join(servers, ","))
		if err != nil {
			return fmt.Errorf("failed to convert DNS servers to UTF16: %w", err)
		}

		settings.NameServer = nameServer
		settings.Flags |= dnsSettingNameserver

		if ipv6 {
			settings.Flags |= dnsSettingIpv6
		}
	}

	ret, _, _ = procSetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&settings)),
	)
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to set DNS settings: %w", windows.Errno(ret))
	}

	return nil
}

func (a *SwiftInterface) AddRoute(route netlink.Route) error {
	// TODO: code it
	return errors.New("not implemented")
}

func (a *SwiftInterface) SetStatus(status swiftypes.InterfaceStatus) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	var ifRow windows.MibIfRow

	ifRow.Index = uint32(index)

	ret, _, _ := procGetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to retrieve interface entry: %w", err)
	}

	switch status {
	case swiftypes.InterfaceUp:
		ifRow.OperStatus = windows.IfOperStatusUp
	case swiftypes.InterfaceDown:
		ifRow.OperStatus = windows.IfOperStatusDown
	}

	ret, _, _ = procSetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to set interface status: %w", err)
	}

	return nil
}
