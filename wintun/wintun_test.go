//go:build windows

package wintun

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
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

	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return uint16(^sum)
}

var testGUID = swiftypes.GUID{Data1: 0x12345678, Data2: 0x9ABC, Data3: 0xDEF0, Data4: [8]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xEF, 0x01}}
var adapterName = "TestWintunAdapter"
var tunnelType = "TestTunnel"

func TestWintunAdapter(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	if adapter.name != adapterName {
		t.Errorf("Expected adapter name %s, got %s", adapterName, adapter.name)
	}

	session, err := adapter.StartSession(0x400000)
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close()

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

	checksum := calculateChecksum(packet[20:])
	binary.BigEndian.PutUint16(packet[24:], checksum)

	binary.BigEndian.PutUint16(packet[2:], uint16(len(packet)))

	ipChecksum := calculateChecksum(packet[:20])
	binary.BigEndian.PutUint16(packet[10:], ipChecksum)

	for i := 0; i < 10; i++ {
		if _, err := session.Write(packet); err != nil {
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

	session, err := adapter.StartSession(0x400000)
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close()

	buf := make([]byte, 1500)

	if n, err := session.ReadNow(buf); err != nil && !errors.Is(err, ErrNoDataAvailable) {
		t.Errorf("Failed to receive packet: %v", err)
	} else {
		t.Logf("Received packet: %v", hex.EncodeToString(buf[:n]))
	}
}

func TestSession_ReceivePacket(t *testing.T) {
	adapter, err := NewWintunAdapterWithGUID(adapterName, tunnelType, testGUID)
	if err != nil {
		t.Fatalf("Failed to create Wintun adapter: %v", err)
	}
	defer adapter.Close()

	session, err := adapter.StartSession(0x400000)
	if err != nil {
		t.Fatalf("Failed to start Wintun session: %v", err)
	}
	defer session.Close()

	buf := make([]byte, 1500)

	if n, err := session.Read(buf); err != nil {
		t.Errorf("Failed to receive packet: %v", err)
	} else {
		t.Logf("Received packet: %v", hex.EncodeToString(buf[:n]))
	}
}
