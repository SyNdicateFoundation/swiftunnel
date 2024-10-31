//go:build !darwin && !linux && !solaris && !freebsd && !windows

package gateway

import (
	"errors"
	"net"
)

var errNotImplemented = errors.New("not implemented")

func DiscoverGatewayIPv4() (ip net.IP, err error) {
	return ip, errNotImplemented
}

func DiscoverGatewayIPv6() (ip net.IP, err error) {
	return ip, errNotImplemented
}
