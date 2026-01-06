//go:build windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
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
	procCreateIpForwardEntry2           = iphlpapi.NewProc("CreateIpForwardEntry2")
	procInitializeIpForwardEntry        = iphlpapi.NewProc("InitializeIpForwardEntry")
	procSetInterfaceDnsSettings         = iphlpapi.NewProc("SetInterfaceDnsSettings")
	procGetInterfaceDnsSettings         = iphlpapi.NewProc("GetInterfaceDnsSettings")
	procDeleteIpForwardEntry2           = iphlpapi.NewProc("DeleteIpForwardEntry2")
	procGetIpForwardTable2              = iphlpapi.NewProc("GetIpForwardTable2")
	procSetIpForwardEntry2              = iphlpapi.NewProc("SetIpForwardEntry2")
	procFreeMibTable                    = iphlpapi.NewProc("FreeMibTable")
	procGetIfEntry                      = iphlpapi.NewProc("GetIfEntry")
	procSetIfEntry                      = iphlpapi.NewProc("SetIfEntry")
)

const (
	mibIPForwardProtoNetMgmt = 3
	nlRouteOriginManual      = 0
)

// GetAdapterLUID retrieves the Locally Unique Identifier of the Windows adapter.
func (a *SwiftInterface) GetAdapterLUID() (swiftypes.LUID, error) {
	if a.service == nil {
		return swiftypes.NilLUID, ErrCannotFindAdapter
	}
	return a.service.GetAdapterLUID()
}

// GetAdapterGUID retrieves the Globally Unique Identifier of the Windows adapter.
func (a *SwiftInterface) GetAdapterGUID() (swiftypes.GUID, error) {
	if a.service == nil {
		return swiftypes.NilGUID, ErrCannotFindAdapter
	}
	return a.service.GetAdapterGUID()
}

// GetAdapterName retrieves the friendly name of the adapter.
func (a *SwiftInterface) GetAdapterName() (string, error) {
	if a.service == nil {
		return "", ErrCannotFindAdapter
	}
	return a.service.GetAdapterName()
}

// GetAdapterIndex retrieves the IPv4 IF index of the adapter.
func (a *SwiftInterface) GetAdapterIndex() (int, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}
	return a.service.GetAdapterIndex()
}

// SetMTU updates the MTU for the adapter using SetIfEntry.
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

// SetUnicastIpAddressEntry assigns an IP address to the adapter LUID.
func (a *SwiftInterface) SetUnicastIpAddressEntry(config *swiftypes.UnicastConfig) error {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return err
	}

	var addressRow windows.MibUnicastIpAddressRow

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

	ones, bits := config.IPNet.Mask.Size()
	if ones > bits {
		return fmt.Errorf("invalid subnet mask: %v", config.IPNet.Mask)
	}

	if config.DadState == windows.IpDadStateInvalid {
		addressRow.DadState = windows.IpDadStatePreferred
	}

	addressRow.InterfaceLuid = luid.ToUint64()
	addressRow.OnLinkPrefixLength = uint8(ones)
	addressRow.DadState = config.DadState

	ret, _, _ := procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) && !errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
		return fmt.Errorf("failed to create unicast IP address config: %w (error code: %d)", errno, ret)
	}

	return nil
}

// SetDNS configures DNS servers and search domains for the interface.
func (a *SwiftInterface) SetDNS(config *swiftypes.DNSConfig) error {
	guid, err := a.GetAdapterGUID()
	if err != nil {
		return err
	}

	var settings dnsInterfaceSettings
	settings.Version = 1

	ret, _, _ := procGetInterfaceDnsSettings.Call(
		uintptr(unsafe.Pointer(&guid)),
		uintptr(unsafe.Pointer(&settings)),
	)
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to get DNS settings: %w", windows.Errno(ret))
	}

	if config.Domain != "" {
		domain, err := windows.UTF16PtrFromString(config.Domain)
		if err != nil {
			return fmt.Errorf("failed to convert domain to UTF16: %w", err)
		}
		settings.Domain = domain
		settings.Flags |= dnsSettingDomain
	}

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

// SetStatus modifies the administrative status of the interface.
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

// AddRoute adds a network route via the current interface.
func (a *SwiftInterface) AddRoute(route *netlink.Route) error {
	row, err := a.routeToRow(route)
	if err != nil {
		return err
	}

	ret, _, _ := procCreateIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
		// If the route already exists, we consider it a success (idempotent).
		if errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
			return nil
		}
		return fmt.Errorf("failed to add route: %w", errno)
	}

	return nil
}

// ReplaceRoute replaces a network route via the current interface.
// On Windows, if the exact route exists, we update it. If not, we create it.
func (a *SwiftInterface) ReplaceRoute(route *netlink.Route) error {
	row, err := a.routeToRow(route)
	if err != nil {
		return err
	}

	// Try to create the route first
	ret, _, _ := procCreateIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
	errno := windows.Errno(ret)

	if errors.Is(errno, windows.ERROR_SUCCESS) {
		return nil
	}

	// If it already exists, update properties (like Metric) using SetIpForwardEntry2
	if errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
		ret, _, _ = procSetIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
		if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
			return fmt.Errorf("failed to replace (set) route: %w", errno)
		}
		return nil
	}

	return fmt.Errorf("failed to replace (create) route: %w", errno)
}

// ChangeRoute changes an existing network route via the current interface.
func (a *SwiftInterface) ChangeRoute(route *netlink.Route) error {
	row, err := a.routeToRow(route)
	if err != nil {
		return err
	}

	// SetIpForwardEntry2 modifies properties of an existing route.
	ret, _, _ := procSetIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
		return fmt.Errorf("failed to change route: %w", errno)
	}

	return nil
}

// AppendRoute appends a network route via the current interface.
// On Windows, this is functionally equivalent to AddRoute (CreateIpForwardEntry2).
func (a *SwiftInterface) AppendRoute(route *netlink.Route) error {
	row, err := a.routeToRow(route)
	if err != nil {
		return err
	}

	ret, _, _ := procCreateIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
		// Append in netlink often implies adding a multipath route, which Windows supports via Create.
		if errors.Is(errno, windows.ERROR_OBJECT_ALREADY_EXISTS) {
			// If exact match exists, we consider it a success for Append/Add logic typically
			return nil
		}
		return fmt.Errorf("failed to append route: %w", errno)
	}

	return nil
}

// RouteList retrieves network routes via the current interface.
func (a *SwiftInterface) RouteList(family int) ([]netlink.Route, error) {
	idx, err := a.GetAdapterIndex()
	if err != nil {
		return nil, err
	}

	// family: windows.AF_INET (2) or windows.AF_INET6 (23) or AF_UNSPEC (0)
	var table *windows.MibIpForwardTable2
	ret, _, _ := procGetIpForwardTable2.Call(
		uintptr(family),
		uintptr(unsafe.Pointer(&table)),
	)
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
		return nil, fmt.Errorf("failed to get route table: %w", errno)
	}

	defer procFreeMibTable.Call(uintptr(unsafe.Pointer(table)))

	if table.NumEntries == 0 {
		return []netlink.Route{}, nil
	}

	var routes []netlink.Route
	rowSize := unsafe.Sizeof(windows.MibIpForwardRow2{})
	// table.Table is an array starting at offset 4 (after NumEntries uint32) + padding
	base := uintptr(unsafe.Pointer(&table.Table[0]))

	for i := uint32(0); i < table.NumEntries; i++ {
		//goland:noinspection GoVetUnsafePointer
		row := (*windows.MibIpForwardRow2)(unsafe.Pointer(base + uintptr(i)*rowSize))

		// Filter by interface index
		if int(row.InterfaceIndex) != idx {
			continue
		}

		nlRoute := netlink.Route{
			LinkIndex: idx,
			Priority:  int(row.Metric),
			Protocol:  netlink.RouteProtocol(row.Protocol),
		}

		// Parse Destination
		if row.DestinationPrefix.PrefixLength > 0 {
			f := row.DestinationPrefix.Prefix.Family
			var ip net.IP
			if f == windows.AF_INET {
				b := (*[4]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix.Data[0]))
				ip = net.IPv4(b[0], b[1], b[2], b[3])
			} else if f == windows.AF_INET6 {
				b := (*[16]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix.Data[1]))
				ip = b[:]
			}
			if ip != nil {
				nlRoute.Dst = &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(int(row.DestinationPrefix.PrefixLength), 8*len(ip)),
				}
			}
		}

		// Parse Gateway (NextHop)
		if row.NextHop.Family != 0 {
			f := row.NextHop.Family
			var gw net.IP
			if f == windows.AF_INET {
				b := (*[4]byte)(unsafe.Pointer(&row.NextHop.Data[0]))
				gw = net.IPv4(b[0], b[1], b[2], b[3])
			} else if f == windows.AF_INET6 {
				b := (*[16]byte)(unsafe.Pointer(&row.NextHop.Data[1]))
				gw = b[:]
			}
			nlRoute.Gw = gw
		}

		routes = append(routes, nlRoute)
	}

	return routes, nil
}

// RemoveRoute removes a network route via the current interface.
func (a *SwiftInterface) RemoveRoute(route *netlink.Route) error {
	row, err := a.routeToRow(route)
	if err != nil {
		return err
	}

	// DeleteIpForwardEntry2 finds the entry matching Interface, Destination, and NextHop.
	ret, _, _ := procDeleteIpForwardEntry2.Call(uintptr(unsafe.Pointer(row)))
	if errno := windows.Errno(ret); !errors.Is(errno, windows.ERROR_SUCCESS) {
		// Optionally ignore "element not found" (ERROR_FILE_NOT_FOUND or similar) if idempotency is desired.
		return fmt.Errorf("failed to remove route: %w", errno)
	}

	return nil
}

// routeToRow converts a netlink.Route to a Windows MIB_IPFORWARD_ROW2 structure.
func (a *SwiftInterface) routeToRow(route *netlink.Route) (*windows.MibIpForwardRow2, error) {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return nil, err
	}

	var row windows.MibIpForwardRow2
	ret, _, _ := procInitializeIpForwardEntry.Call(uintptr(unsafe.Pointer(&row)))
	if err := windows.Errno(ret); !errors.Is(err, windows.ERROR_SUCCESS) {
		return nil, fmt.Errorf("failed to initialize ip forward entry: %w", err)
	}

	row.InterfaceLuid = luid.ToUint64()
	row.InterfaceIndex = 0 // LUID takes precedence
	row.Metric = uint32(route.Priority)
	row.Protocol = mibIPForwardProtoNetMgmt
	row.Origin = nlRouteOriginManual
	row.ValidLifetime = 0xffffffff
	row.PreferredLifetime = 0xffffffff

	// Set Destination Prefix
	if route.Dst != nil {
		ones, _ := route.Dst.Mask.Size()
		row.DestinationPrefix.PrefixLength = uint8(ones)

		if ipv4 := route.Dst.IP.To4(); ipv4 != nil {
			row.DestinationPrefix.Prefix.Family = windows.AF_INET
			copy((*[4]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix.Data[0]))[:], ipv4)
		} else if ipv6 := route.Dst.IP.To16(); ipv6 != nil {
			row.DestinationPrefix.Prefix.Family = windows.AF_INET6
			copy((*[16]byte)(unsafe.Pointer(&row.DestinationPrefix.Prefix.Data[1]))[:], ipv6)
		}
	} else {
		// Handle default route logic
		if route.Gw != nil {
			if route.Gw.To4() != nil {
				row.DestinationPrefix.Prefix.Family = windows.AF_INET
				row.DestinationPrefix.PrefixLength = 0
			} else {
				row.DestinationPrefix.Prefix.Family = windows.AF_INET6
				row.DestinationPrefix.PrefixLength = 0
			}
		} else {
			return nil, errors.New("cannot determine address family for default route (dst=nil, gw=nil)")
		}
	}

	// Set Next Hop (Gateway)
	if route.Gw != nil {
		if ipv4 := route.Gw.To4(); ipv4 != nil {
			row.NextHop.Family = windows.AF_INET
			copy((*[4]byte)(unsafe.Pointer(&row.NextHop.Data[0]))[:], ipv4)
		} else if ipv6 := route.Gw.To16(); ipv6 != nil {
			row.NextHop.Family = windows.AF_INET6
			copy((*[16]byte)(unsafe.Pointer(&row.NextHop.Data[1]))[:], ipv6)
		}
	}

	return &row, nil
}
