// main_test.go
package wintun

import (
	"github.com/vishvananda/netlink"
	"testing"
	_ "time"
)

func TestWintunAdapter(t *testing.T) {
	adapterName := "TestWintunAdapter"
	tunnelType := "TestTunnel"
	
	// Create a new Wintun adapter
	adapter, err := NewWintunAdapter(adapterName, tunnelType)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close() // Ensure the adapter is closed after the test
	
	// Check the adapter name
	if adapter.Name != adapterName {
		t.Errorf("Expected adapter name %s, got %s", adapterName, adapter.Name)
	}
	
	// Start a session
	session, err := adapter.StartSession(2048) // Example capacity
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close() // Ensure the session is closed after the test
	
	// Send a packet
	packet := []byte("Test packet data")
	if err := session.SendPacket(packet); err != nil {
		t.Errorf("Failed to send packet: %v", err)
	}
	
	// Receive a packet
	receivedPacket, err := session.ReceivePacket()
	if err != nil {
		t.Errorf("Failed to receive packet: %v", err)
	}
	
	// Validate the received packet
	if string(receivedPacket) != string(packet) {
		t.Errorf("Expected received packet %s, got %s", string(packet), string(receivedPacket))
	}
}

func TestGetRunningDriverVersion(t *testing.T) {
	adapterName := "TestWintunAdapter"
	tunnelType := "TestTunnel"
	
	// Create a new Wintun adapter
	adapter, err := NewWintunAdapter(adapterName, tunnelType)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()
	
	// Get the running driver version
	version, err := adapter.GetRunningDriverVersion()
	if err != nil {
		t.Errorf("Failed to get running Wintun driver version: %v", err)
	} else {
		t.Logf("Running Wintun driver version: %s", version)
	}
}

func TestGetAdapterLUID(t *testing.T) {
	adapterName := "TestWintunAdapter"
	tunnelType := "TestTunnel"
	
	// Create a new Wintun adapter
	adapter, err := NewWintunAdapter(adapterName, tunnelType)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()
	
	// Get the adapter LUID
	luid, err := adapter.GetAdapterLUID()
	if err != nil {
		t.Errorf("Failed to get adapter LUID: %v", err)
	} else {
		t.Logf("Adapter LUID: %v", luid)
	}
}

func TestAddressManagement(t *testing.T) {
	adapterName := "TestWintunAdapter"
	tunnelType := "TestTunnel"
	
	// Create a new Wintun adapter
	adapter, err := NewWintunAdapter(adapterName, tunnelType)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()
	
	// Example address to add and delete (update with a valid address)
	addr, err := netlink.ParseAddr("237.84.2.178/24")
	if err != nil {
		t.Errorf("Failed to parse IP address: %v", err)
	}
	
	// Add address
	if err := adapter.AddrAdd(addr); err != nil {
		t.Errorf("Failed to add address: %v", err)
	}
	
	// List addresses
	addresses, err := adapter.AddrList()
	if err != nil {
		t.Errorf("Failed to list addresses: %v", err)
	} else if len(addresses) == 0 {
		t.Errorf("Expected at least one address, got none")
	}
	
	// Delete address
	if err := adapter.AddrDel(addr); err != nil {
		t.Errorf("Failed to delete address: %v", err)
	}
}
