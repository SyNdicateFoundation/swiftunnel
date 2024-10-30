// main_test.go
package wintungo

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"golang.org/x/sys/windows"
	"net"
	"testing"
)

func calculateChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 + uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	// Add overflow
	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return uint16(^sum)
}

var testGUID = windows.GUID{Data1: 0x12345678, Data2: 0x9ABC, Data3: 0xDEF0, Data4: [8]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xEF, 0x01}}
var adapterName = "TestWintunAdapter"
var tunnelType = "TestTunnel"

func TestWintunAdapter(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close() // Ensure the adapter is closed after the test

	// Check the adapter name
	if adapter.name != adapterName {
		t.Errorf("Expected adapter name %s, got %s", adapterName, adapter.name)
	}

	// Start a session
	session, err := adapter.StartSession(0x400000) // Example capacity
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close() // Ensure the session is closed after the test

	packet := []byte{
		0x45, 0x00, 0x00, 0x38, // IPv4 Header: Version, IHL, Type of Service, Total Length
		0x00, 0x00, 0x40, 0x00, // Identification, Flags, Fragment Offset
		0x40, 0x01, 0x00, 0x00, // Time to Live (64), Protocol (ICMP), Header Checksum (to be calculated)
		0xC0, 0xA8, 0x01, 0x01, // Source IP: 192.168.1.1
		0x08, 0x08, 0x08, 0x08, // Destination IP: 8.8.8.8
		0x08, 0x00, 0x00, 0x00, // ICMP Type (Echo Request), Code, Checksum (to be calculated), Identifier, Sequence Number
		0x00, 0x01, // Identifier
		0x00, 0x01, // Sequence Number
		0x48, 0x65, 0x6c, 0x6c, // Payload ("Hello, ICMP!")
		0x6f, 0x2c, 0x20, 0x49,
		0x43, 0x4d, 0x50, 0x21,
	}

	// Calculate checksum for ICMP header
	checksum := calculateChecksum(packet[20:])        // ICMP header starts after the IPv4 header (20 bytes)
	binary.BigEndian.PutUint16(packet[24:], checksum) // Update the checksum in the packet

	// Update IPv4 header total length
	binary.BigEndian.PutUint16(packet[2:], uint16(len(packet)))

	// Update IPv4 header checksum
	ipChecksum := calculateChecksum(packet[:20])        // Calculate the checksum for the IP header
	binary.BigEndian.PutUint16(packet[10:], ipChecksum) // Update the checksum in the IP header

	for i := 0; i < 10; i++ {
		if err := session.SendPacket(packet); err != nil {
			t.Errorf("Failed to send packet: %v", err)
		}
	}

}

func TestGetRunningDriverVersion(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
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
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
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

func TestAdapter_GetAdapterGUID(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	// Get the adapter GUID
	guid, err := adapter.GetAdapterGUID()
	if err != nil {
		t.Errorf("Failed to get adapter GUID: %v", err)
	} else {
		t.Logf("Wanted GUID: %v", testGUID)
		t.Logf("Current GUID: %v", guid)
	}
}

func TestAdapter_GetAdapterIndex(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	// Get the adapter index
	index, err := adapter.GetAdapterIndex()
	if err != nil {
		t.Errorf("Failed to get adapter index: %v", err)
	} else {
		t.Logf("Adapter index: %v", index)
	}
}

func TestSession_ReceivePacketNow(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	session, err := adapter.StartSession(0x400000) // Example capacity
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close()

	packet, err := session.ReceivePacketNow()
	if err != nil && !errors.Is(err, ErrNoDataAvailable) {
		t.Errorf("Failed to receive packet: %v", err)
	} else {
		t.Logf("Received packet: %v", packet)
	}
}

func TestSession_ReceivePacket(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	session, err := adapter.StartSession(0x400000) // Example capacity
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close()

	packet, err := session.ReceivePacket()
	if err != nil {
		t.Errorf("Failed to receive packet: %v", err)
	} else {
		t.Logf("Received packet: %v", hex.EncodeToString(packet))
	}
}

func TestAdapter_SetMTU(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	// Set the MTU to 1400
	if err := adapter.SetMTU(1400); err != nil {
		t.Errorf("Failed to set MTU: %v", err)
	}
}

func TestAdapter_SetUnicastIpAddressEntry(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	// Set the IP address
	_, ipNet, _ := net.ParseCIDR("10.6.7.7/24")
	if err := adapter.SetUnicastIpAddressEntry(ipNet, IpDadStatePreferred); err != nil {
		t.Errorf("Failed to set unicast IP address: %v", err)
	}
}

func TestAdapter_SetDNS(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}

	defer adapter.Close()

	// Set the DNS servers
	config := DNSConfig{
		Domain: "example.com",
		DnsServers: []string{
			"8.8.8.8",
			"8.8.4.4",
		},
	}
	if err := adapter.SetDNS(config); err != nil {
		t.Errorf("Failed to set DNS servers: %v", err)
	}
}
