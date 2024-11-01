//go:build windows

package Swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/Swiftunnel/swiftypes"
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

func setUnicastIpAddressEntry(luid swiftypes.LUID, entry *net.IPNet, dadState DadState) error {
	var addressRow mibUnicastIPAddressRow

	procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))

	if ipv4 := entry.IP.To4(); ipv4 != nil {
		addressRow.Address.Family = windows.AF_INET
		copy(addressRow.Address.Data[:], ipv4)
	} else {
		addressRow.Address.Family = windows.AF_INET6
		copy(addressRow.Address.Data[:], entry.IP.To16())
	}

	ones, _ := entry.Mask.Size()
	addressRow.InterfaceLUID = luid.ToUint64()
	addressRow.OnLinkPrefixLength = uint8(ones)
	addressRow.DadState = dadState

	ret, _, _ := procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) && !errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
		return fmt.Errorf("failed to create unicast IP address entry: %w", errno)
	}

	return nil
}

func setDNS(guid swiftypes.GUID, config *swiftypes.DNSConfig) error {
	var settings dnsInterfaceSettings

	settings.Version = 1
	settings.Flags = 0

	ret, _, _ := procGetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&settings)),
	)
	if ret != 0 {
		return windows.Errno(ret)
	}

	if config.Domain != "" {
		domain, err := windows.UTF16PtrFromString(config.Domain)
		if err != nil {
			return err
		}
		settings.Domain = domain
	}

	if len(config.DnsServers) > 0 {
		var servers strings.Builder
		for _, server := range config.DnsServers {
			servers.WriteString(server.String())
		}

		fromString, err := windows.UTF16PtrFromString(servers.String())
		if err != nil {
			return err
		}

		settings.NameServer = fromString
	}

	ret, _, _ = procSetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&settings)),
	)
	if ret != 0 {
		return windows.Errno(ret)
	}

	return nil
}
