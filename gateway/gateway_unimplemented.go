//go:build !darwin && !linux && !windows

package gateway

import (
	"errors"
	"net"
)

var errNotImplemented = errors.New("not implemented")

// DiscoverGatewayIPv4 is a stub for unsupported platforms.
func DiscoverGatewayIPv4() (ip net.IP, err error) {
	return ip, errNotImplemented
}

// DiscoverGatewayIPv6 is a stub for unsupported platforms.
func DiscoverGatewayIPv6() (ip net.IP, err error) {
	return ip, errNotImplemented
}
