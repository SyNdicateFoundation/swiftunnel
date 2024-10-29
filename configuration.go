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

// nlDadState represents the duplicate address detection state.
type nlDadState uint32

const (
	IpDadStateInvalid    nlDadState = iota // 0: The DAD state is invalid.
	IpDadStateTentative                    // 1: The DAD state is tentative.
	IpDadStateDuplicate                    // 2: A duplicate IP address has been detected.
	IpDadStateDeprecated                   // 3: The IP address has been deprecated.
	IpDadStatePreferred                    // 4: The IP address is preferred.
)

type DNSConfig struct {
	Domain     string
	DnsServers []string
}

type sockaddrInet struct {
	Ipv4     windows.RawSockaddrInet4
	Ipv6     windows.RawSockaddrInet6
	siFamily uint16 // Address family (AF_INET for IPv4, AF_INET6 for IPv6)
}

type mibUnicastipaddressRow struct {
	Address            sockaddrInet
	InterfaceLuid      windows.LUID
	OnLinkPrefixLength uint8
	DadState           nlDadState
}

type dnsInterfaceSettings struct {
	Version             uint32
	Flags               uint64
	Domain              *uint16
	NameServer          *uint16
	SearchList          *uint16
	RegistrationEnabled uint32
	RegisterAdapterName uint32
	EnableLLMNR         uint32
	QueryAdapterName    uint32
	ProfileNameServer   *uint16
}

const (
	maxInterfaceNameLen = 256
	maxlenPhysaddr      = 8
	maxlenIfdescr       = 256
)

// mibIfrow is the Go representation of the mibIfrow structure.
type mibIfrow struct {
	WszName           [maxInterfaceNameLen]uint16 // WCHAR is equivalent to uint16 in Go
	DwIndex           uint32                      // IF_INDEX (equivalent to DWORD in Go)
	DwType            uint32                      // IFTYPE (also DWORD)
	DwMtu             uint32
	DwSpeed           uint32
	DwPhysAddrLen     uint32
	BPhysAddr         [maxlenPhysaddr]byte
	DwAdminStatus     uint32
	DwOperStatus      uint32 // INTERNAL_IF_OPER_STATUS (equivalent to DWORD in Go)
	DwLastChange      uint32
	DwInOctets        uint32
	DwInUcastPkts     uint32
	DwInNUcastPkts    uint32
	DwInDiscards      uint32
	DwInErrors        uint32
	DwInUnknownProtos uint32
	DwOutOctets       uint32
	DwOutUcastPkts    uint32
	DwOutNUcastPkts   uint32
	DwOutDiscards     uint32
	DwOutErrors       uint32
	DwOutQLen         uint32
	DwDescrLen        uint32
	BDescr            [maxlenIfdescr]byte
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

	var addressRow mibUnicastipaddressRow

	// Initialize the entry
	ret, _, _ := procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to initialize unicast IP address entry: %w", errno)
	}

	// Set the IP address and family
	if ipv4 := entry.IP.To4(); ipv4 != nil {
		addressRow.Address.Ipv4.Family = windows.AF_INET
		copy(addressRow.Address.Ipv4.Addr[:], ipv4)
	} else {
		addressRow.Address.Ipv6.Family = windows.AF_INET6
		copy(addressRow.Address.Ipv6.Addr[:], entry.IP.To16())
	}

	// Set additional fields
	ones, _ := entry.Mask.Size()
	addressRow.InterfaceLuid = luid
	addressRow.OnLinkPrefixLength = uint8(ones)
	addressRow.DadState = dadState

	// Create the unicast IP address entry
	ret, _, _ = procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))
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
