package swiftutils

import "net"

func IsIPv4(packet []byte) bool {
	return 4 == (packet[0] >> 4)
}

func IsIPv6(packet []byte) bool {
	return 6 == (packet[0] >> 4)
}

func IPv4Destination(packet []byte) net.IP {
	return net.IPv4(packet[16], packet[17], packet[18], packet[19])
}

func IPv6Destination(packet []byte) net.IP {
	return packet[16:32]
}
