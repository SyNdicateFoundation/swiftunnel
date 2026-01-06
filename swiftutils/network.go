package swiftutils

import (
	"net"
)

// IsIPv4 returns true if the packet header indicates IPv4.
func IsIPv4(packet []byte) bool {
	if len(packet) < 1 {
		return false
	}
	return 4 == (packet[0] >> 4)
}

// IsIPv6 returns true if the packet header indicates IPv6.
func IsIPv6(packet []byte) bool {
	if len(packet) < 1 {
		return false
	}

	return 6 == (packet[0] >> 4)
}

// IPv4Destination extracts the destination IP address from an IPv4 packet.
func IPv4Destination(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}

	if packet[0]>>4 != 4 {
		return nil
	}

	dest := make(net.IP, 4)
	copy(dest, packet[16:20])

	return dest
}

// IPv6Destination extracts the destination IP address from an IPv6 packet.
func IPv6Destination(packet []byte) net.IP {
	if len(packet) < 40 {
		return nil
	}

	return packet[24:40]
}

// IPDestination extracts the destination IP address based on the packet version.
func IPDestination(packet []byte) net.IP {
	if IsIPv6(packet) {
		return IPv6Destination(packet)
	}

	return IPv4Destination(packet)
}

// IPSource extracts the source IP address based on the packet version.
func IPSource(packet []byte) net.IP {
	if IsIPv6(packet) {
		return IPv6Source(packet)
	}

	return IPv4Source(packet)
}

// IPv4Source extracts the source IP address from an IPv4 packet.
func IPv4Source(packet []byte) net.IP {
	if len(packet) < 20 {
		return nil
	}

	return net.IPv4(packet[12], packet[13], packet[14], packet[15])
}

// IPv6Source extracts the source IP address from an IPv6 packet.
func IPv6Source(packet []byte) net.IP {
	if len(packet) < 40 {
		return nil
	}

	return packet[8:24]
}

// IPv4HeaderLength calculates the length of the IPv4 header including options.
func IPv4HeaderLength(packet []byte) int {
	if len(packet) < 20 {
		return 0
	}

	headerLength := (packet[0] & 0x0F) * 4
	return int(headerLength)
}

// IPv4Protocol retrieves the protocol byte from an IPv4 header.
func IPv4Protocol(packet []byte) int {
	if len(packet) < 20 {
		return 0
	}

	return int(packet[9])
}

// IPv6NextHeader retrieves the Next Header byte from an IPv6 header.
func IPv6NextHeader(packet []byte) int {
	if len(packet) < 40 {
		return 0
	}

	return int(packet[6])
}

// ValidateIPv4 checks if the packet length matches the IPv4 header specifications.
func ValidateIPv4(packet []byte) bool {
	if len(packet) < 20 {
		return false
	}

	ihl := (packet[0] & 0x0F) * 4
	totalLen := int(packet[2])<<8 | int(packet[3])

	if int(ihl) > len(packet) || totalLen > len(packet) || ihl < 20 {
		return false
	}

	return true
}
