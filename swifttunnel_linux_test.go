//go:build linux

package swiftunnel

import (
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"net"
	"testing"
)

func TestNewSwiftInterface(t *testing.T) {
	config := Config{
		AdapterName: "tun0",
		AdapterType: swiftypes.AdapterTypeTUN,
		MTU:         1500,
		UnicastIP:   &net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.CIDRMask(24, 32)},
	}

	// Create a new SwiftInterface
	adapter, err := NewSwiftInterface(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer adapter.Close()

	// Check if the adapter name is set correctly
	name, err := adapter.GetAdapterName()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if name != config.AdapterName {
		t.Errorf("expected adapter name %s, got %s", config.AdapterName, name)
	}

	// Check if the adapter index can be retrieved
	index, err := adapter.GetAdapterIndex()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if index == 0 {
		t.Error("expected a valid adapter index, got 0")
	}

}

func TestSetMTU(t *testing.T) {
	config := Config{
		AdapterName: "tun0",
		AdapterType: swiftypes.AdapterTypeTUN,
	}

	adapter, err := NewSwiftInterface(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer adapter.Close()

	if err := adapter.SetMTU(1400); err != nil {
		t.Fatalf("expected no error setting MTU, got %v", err)
	}

}

func TestSetUnicastIpAddressEntry(t *testing.T) {
	config := Config{
		AdapterName: "tun0",
		AdapterType: swiftypes.AdapterTypeTUN,
	}

	adapter, err := NewSwiftInterface(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer adapter.Close()

	ipNet := net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.CIDRMask(24, 32)}
	if err := adapter.SetUnicastIpAddressEntry(&ipNet); err != nil {
		t.Fatalf("expected no error setting IP address, got %v", err)
	}
}
