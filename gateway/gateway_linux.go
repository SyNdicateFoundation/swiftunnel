//go:build linux

package gateway

import (
	"errors"
	"net"
	"os/exec"
	"strings"
)

var ErrCantParse = errors.New("unable to parse gateway IP from route output")

// DiscoverGatewayIPv4 finds the IPv4 default gateway for Linux.
func DiscoverGatewayIPv4() (net.IP, error) {
	return discoverGateway("ip route | grep 'default' | awk '{print $3}'")
}

// DiscoverGatewayIPv6 finds the IPv6 default gateway for Linux.
func DiscoverGatewayIPv6() (net.IP, error) {
	return discoverGateway("ip -6 route | grep 'default' | awk '{print $3}'")
}

// discoverGateway executes a shell command to fetch and parse the gateway IP address.
func discoverGateway(cmdStr string) (net.IP, error) {
	output, err := exec.Command("sh", "-c", cmdStr).CombinedOutput()
	if err != nil {
		return nil, err
	}

	ipStr := strings.TrimSpace(string(output))
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, ErrCantParse
	}
	return ip, nil
}
