package wintungo

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"net"
	"strings"
	"unsafe"
)

// Lazy-loaded DLL for Windows IP helper functions
var (
	iphlpapi                            = windows.NewLazySystemDLL("iphlpapi.dll")
	convertLUIDToIndex                  = iphlpapi.NewProc("ConvertInterfaceLuidToIndex")
	convertInterfaceLuidToGuid          = iphlpapi.NewProc("ConvertInterfaceLuidToGuid")
	procCreateUnicastIpAddressEntry     = iphlpapi.NewProc("CreateUnicastIpAddressEntry")
	procInitializeUnicastIpAddressEntry = iphlpapi.NewProc("InitializeUnicastIpAddressEntry")
	procSetInterfaceDnsSettings         = iphlpapi.NewProc("SetInterfaceDnsSettings")
	procGetInterfaceDnsSettings         = iphlpapi.NewProc("GetInterfaceDnsSettings")
	procGetIfEntry                      = iphlpapi.NewProc("GetIfEntry")
	procSetIfEntry                      = iphlpapi.NewProc("SetIfEntry")
)

type DNSConfig struct {
	Domain     string
	DnsServers []string
}

// SetMTU sets the MTU for the adapter using a netsh command
func (a *Adapter) SetMTU(mtu uint32) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	var ifRow mibIfrow

	ifRow.DwIndex = index // Adapter index, you may need to retrieve it using GetAdaptersInfo or similar

	// Get the interface entry to modify the MTU
	ret, _, err := procGetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if ret != 0 {
		return fmt.Errorf("failed to retrieve interface entry: %w", err)
	}

	// Set the MTU and update the interface
	ifRow.DwMtu = mtu
	ret, _, err = procSetIfEntry.Call(uintptr(unsafe.Pointer(&ifRow)))
	if ret != 0 {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	return nil
}

func (a *Adapter) SetUnicastIpAddressEntry(entry *net.IPNet, dadState nlDadState) error {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return fmt.Errorf("failed to get adapter LUID: %w", err)
	}

	var addressRow nibUnicastIPAddressRow

	procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))

	// Set the IP address and family
	if ipv4 := entry.IP.To4(); ipv4 != nil {
		addressRow.Address.Family = windows.AF_INET
		copy(addressRow.Address.data[:], ipv4)
	} else {
		addressRow.Address.Family = windows.AF_INET6
		copy(addressRow.Address.data[:], entry.IP.To16())
	}

	// Set additional fields
	ones, _ := entry.Mask.Size()
	addressRow.InterfaceLUID = convertLUIDtouint64(luid)
	addressRow.OnLinkPrefixLength = uint8(ones)
	addressRow.DadState = dadState

	// Create the unicast IP address entry
	ret, _, _ := procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) && !errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
		return fmt.Errorf("failed to create unicast IP address entry: %w", errno)
	}

	return nil
}

// SetDNS sets the DNS servers for the adapter
func (a *Adapter) SetDNS(config DNSConfig) error {
	guid, err := a.GetAdapterGUID()
	if err != nil {
		return err
	}

	var settings dnsInterfaceSettings

	// Initialize the settings
	settings.Version = 1 // Set to the current version expected by the API
	settings.Flags = 0   // Adjust flags as needed

	// Retrieve current DNS settings
	ret, _, _ := procGetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),     // Pass GUID directly
		uintptr(unsafe.Pointer(&settings)), // Pointer to settings
	)
	if ret != 0 {
		return windows.Errno(ret)
	}

	// Set the DNS domain if provided
	if config.Domain != "" {
		domain, err := windows.UTF16PtrFromString(config.Domain)
		if err != nil {
			return err
		}
		settings.Domain = domain
	}

	// Prepare DNS servers
	if len(config.DnsServers) > 0 {
		fromString, err := windows.UTF16PtrFromString(strings.Join(config.DnsServers, ","))
		if err != nil {
			return err
		}
		settings.NameServer = fromString
	}

	// Set the new DNS settings
	ret, _, _ = procSetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),     // Pass GUID directly
		uintptr(unsafe.Pointer(&settings)), // Pointer to settings
	)
	if ret != 0 {
		return windows.Errno(ret)
	}

	return nil
}
