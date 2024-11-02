//go:build windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"golang.org/x/sys/windows"
	"log"
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

func setMTU(index uint32, mtu int) error {
	var ifRow windows.MibIfRow

	ifRow.Index = index

	ret, _, err := procGetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if ret != 0 {
		return fmt.Errorf("failed to retrieve interface entry: %w", err)
	}

	ifRow.Mtu = uint32(mtu)
	ret, _, err = procSetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if ret != 0 {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

func setUnicastIpAddressEntry(luid swiftypes.LUID, config *swiftypes.UnicastConfig) error {
	var addressRow mibUnicastIPAddressRow

	// Initialize the addressRow structure
	procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))

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

	log.Println("Successfully created unicast IP address config.")
	return nil
}

func setDNS(guid swiftypes.GUID, config *swiftypes.DNSConfig) error {
	var settings dnsInterfaceSettings

	settings.Version = 1

	// Retrieve the current DNS interface settings
	ret, _, _ := procGetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&settings)),
	)
	if ret != 0 {
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
	if ret != 0 {
		return fmt.Errorf("failed to set DNS settings: %w", windows.Errno(ret))
	}

	return nil
}
