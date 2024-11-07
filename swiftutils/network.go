package swiftutils

import (
	"net"
)

// IsIPv4 checks if the packet is IPv4 based on the first byte.
func IsIPv4(packet []byte) bool {
	if len(packet) < 1 {
		return false
	}
	return 4 == (packet[0] >> 4)
}

// IsIPv6 checks if the packet is IPv6 based on the first byte.
func IsIPv6(packet []byte) bool {
	if len(packet) < 1 {
		return false
	}
	return 6 == (packet[0] >> 4)
}

// IPv4Destination extracts the destination IPv4 address from the packet.
// Assumes IPv4 and at least 20 bytes.
func IPv4Destination(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}
	return net.IPv4(packet[16], packet[17], packet[18], packet[19])
}

// IPv6Destination extracts the destination IPv6 address from the packet.
// Assumes IPv6 and at least 40 bytes.
func IPv6Destination(packet []byte) net.IP {
	if len(packet) < 40 {
		return nil
	}
	// Corrected: Destination IPv6 address starts at byte 24 and ends at byte 39
	return packet[24:40]
}

func IPDestination(packet []byte) net.IP {
	if IsIPv6(packet) {
		return IPv6Destination(packet)
	}

	return IPv4Destination(packet)
}

func IPSource(packet []byte) net.IP {
	if IsIPv6(packet) {
		return IPv6Source(packet)
	}

	return IPv4Source(packet)
}

// IPv4Source extracts the source IPv4 address from the packet.
// Assumes IPv4 and at least 20 bytes.
func IPv4Source(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}
	return net.IPv4(packet[12], packet[13], packet[14], packet[15])
}

// IPv6Source extracts the source IPv6 address from the packet.
// Assumes IPv6 and at least 40 bytes.
func IPv6Source(packet []byte) net.IP {
	if len(packet) < 40 {
		return nil
	}
	// Corrected: Source IPv6 address starts at byte 8 and ends at byte 23
	return packet[8:24]
}

// IPv4HeaderLength returns the length of the IPv4 header, including options.
// It assumes the packet is IPv4 and at least 20 bytes.
func IPv4HeaderLength(packet []byte) int {
	if len(packet) < 20 {
		return 0
	}
	// The header length is encoded in the 4 high bits of the second byte.
	headerLength := (packet[0] & 0x0F) * 4
	return int(headerLength)
}

// IPv4Protocol extracts the protocol field from the IPv4 packet.
// It assumes IPv4 and at least 20 bytes.
func IPv4Protocol(packet []byte) int {
	if len(packet) < 20 {
		return 0
	}
	return int(packet[9])
}

// IPv6NextHeader extracts the next header field from the IPv6 packet.
// It assumes IPv6 and at least 40 bytes.
func IPv6NextHeader(packet []byte) int {
	if len(packet) < 40 {
		return 0
	}
	return int(packet[6])
}
