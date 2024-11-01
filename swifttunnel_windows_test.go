//go:build windows

package swiftunnel

import (
	"net"
	"testing"

	"github.com/XenonCommunity/swiftunnel/swiftypes"
)

func TestNewDefaultConfig(t *testing.T) {
	// Create a default configuration
	config := NewDefaultConfig()

	// Test the default adapter name
	expectedName := "Swiftunnel VPN"
	if config.AdapterName != expectedName {
		t.Errorf("expected AdapterName %s, got %s", expectedName, config.AdapterName)
	}

	// Test the default adapter type name
	expectedTypeName := "Swiftunnel"
	if config.AdapterTypeName != expectedTypeName {
		t.Errorf("expected AdapterTypeName %s, got %s", expectedTypeName, config.AdapterTypeName)
	}

	// Test the default adapter type
	if config.AdapterType != swiftypes.AdapterTypeTUN {
		t.Errorf("expected adapterType AdapterTypeTUN, got %v", config.AdapterType)
	}

	// Test the default driver type
	if config.DriverType != DriverTypeWintun {
		t.Errorf("expected DriverType DriverTypeWintun, got %v", config.DriverType)
	}

	// Test the default MTU
	expectedMTU := 1500
	if config.MTU != expectedMTU {
		t.Errorf("expected MTU %d, got %d", expectedMTU, config.MTU)
	}

	// Test the default DNS servers
	expectedDNSServers := []net.IP{net.IPv4(8, 8, 8, 8), net.IPv4(8, 8, 4, 4)}
	if len(config.DNSConfig.DnsServers) != len(expectedDNSServers) {
		t.Errorf("expected %d DNS servers, got %d", len(expectedDNSServers), len(config.DNSConfig.DnsServers))
	} else {
		for i, expectedDNS := range expectedDNSServers {
			if !config.DNSConfig.DnsServers[i].Equal(expectedDNS) {
				t.Errorf("expected DNS server %s, got %s", expectedDNS, config.DNSConfig.DnsServers[i])
			}
		}
	}

	// Test the default Unicast IP
	expectedUnicastIP := net.IPNet{
		IP:   net.IPv4(10, 18, 21, 1),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	if !config.UnicastIP.IP.Equal(expectedUnicastIP.IP) {
		t.Errorf("expected UnicastIP %s, got %s", expectedUnicastIP.IP, config.UnicastIP.IP)
	}
	if !config.UnicastIP.Mask.String() == expectedUnicastIP.Mask.String() {
		t.Errorf("expected UnicastIP Mask %s, got %s", expectedUnicastIP.Mask.String(), config.UnicastIP.Mask.String())
	}
}
