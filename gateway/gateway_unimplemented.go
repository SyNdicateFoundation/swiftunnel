//go:build !darwin && !linux && !windows && !solaris && !freebsd
// +build !darwin,!linux,!windows,!solaris,!freebsd

package gateway

import (
	"net"
)

func discoverGatewayOSSpecificIPv4() (ip net.IP, err error) {
	return ip, errNotImplemented
}

func discoverGatewayOSSpecificIPv6() (ip net.IP, err error) {
	return ip, errNotImplemented
}
