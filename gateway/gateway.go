package gateway

import (
	"errors"
	"net"
	"runtime"
)

var (
	errNoGateway      = errors.New("no gateway found")
	errCantParse      = errors.New("can't parse string output")
	errNotImplemented = errors.New("not implemented for OS: " + runtime.GOOS)
)

// DiscoverGatewayIPv4 is the OS independent function to get the default ipv4 gateway
func DiscoverGatewayIPv4() (ip net.IP, err error) {
	return discoverGatewayOSSpecificIPv4()
}

// DiscoverGatewayIPv6 is the OS independent function to get the default ipv6 gateway
func DiscoverGatewayIPv6() (ip net.IP, err error) {
	return discoverGatewayOSSpecificIPv6()
}
