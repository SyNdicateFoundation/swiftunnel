package wintungo

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"net"
	"os/exec"
	"syscall"
	"unsafe"
)

// Lazy-loaded DLL for Windows IP helper functions
var (
	iphlpapi                            = windows.NewLazySystemDLL("iphlpapi.dll")
	procCreateUnicastIpAddressEntry     = iphlpapi.NewProc("CreateUnicastIpAddressEntry")
	procInitializeUnicastIpAddressEntry = iphlpapi.NewProc("InitializeUnicastIpAddressEntry")
)

// IpDadStatePreferred Constants for IP address states
const (
	IpDadStatePreferred = 2
)

// sockaddrIn represents an IPv4 socket address structure
type sockaddrIn struct {
	Family uint16
	Port   uint16
	Addr   [4]byte
	Zero   [8]byte // Padding
}

// MibUnicastipaddressRow structure for unicast IP address entries
type MibUnicastipaddressRow struct {
	InterfaceLuid      LUID
	Address            sockaddrIn
	OnLinkPrefixLength uint32
	DadState           uint32
}

// SetMTU sets the MTU for the adapter using a netsh command
func (a *Adapter) SetMTU(mtu uint32) error {
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "subinterface", a.name, fmt.Sprintf("mtu=%d", mtu), "store=persistent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set MTU: %w, output: %s", err, output)
	}
	return nil
}

// SetUnicastIpAddressEntry sets an unicast IP address entry for the adapter
func (a *Adapter) SetUnicastIpAddressEntry(entry *net.IPNet) error {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return fmt.Errorf("failed to get adapter LUID: %w", err)
	}

	var addressRow MibUnicastipaddressRow

	// Initialize the address row
	if _, _, err := procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow))); err != nil && !errors.Is(err, errZero) {
		return fmt.Errorf("failed to initialize unicast IP address entry: %w", err)
	}

	// Set the IP address
	if entry.IP.To4() != nil {
		addressRow.Address.Family = syscall.AF_INET
		copy(addressRow.Address.Addr[:], entry.IP.To4())
	} else {
		return errors.New("only IPv4 addresses are supported")
	}

	// Set additional fields
	ones, _ := entry.Mask.Size()
	addressRow.OnLinkPrefixLength = uint32(ones)
	addressRow.InterfaceLuid = luid
	addressRow.DadState = IpDadStatePreferred

	// Create the unicast IP address entry
	if _, _, err := procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(&addressRow))); err != nil && !errors.Is(err, errZero) {
		return fmt.Errorf("failed to create unicast IP address entry: %w", err)
	}

	return nil
}

// SetDNS sets the DNS servers for the adapter
func (a *Adapter) SetDNS(dnsServers []net.IP) error {
	if len(dnsServers) == 0 {
		return fmt.Errorf("at least one DNS server must be specified")
	}

	// Prepare the commands to set DNS servers
	// First DNS server
	cmdSet := exec.Command("netsh", "interface", "ip", "set", "dns", a.name, "static", dnsServers[0].String())

	if err := cmdSet.Run(); err != nil {
		return fmt.Errorf("failed to set primary DNS server: %w", err)
	}

	// Additional DNS servers
	for i, dns := range dnsServers[1:] {
		cmdAdd := exec.Command("netsh", "interface", "ip", "add", "dns", a.name, dns.String(), "index=", fmt.Sprint(i+2))
		if err := cmdAdd.Run(); err != nil {
			return fmt.Errorf("failed to add DNS server %s: %w", dns.String(), err)
		}
	}

	return nil
}
