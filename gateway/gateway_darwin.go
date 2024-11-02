//go:build darwin

package gateway

import (
	"errors"
	"net"
	"os/exec"
	"strings"
)

var ErrCantParse = errors.New("unable to parse gateway IP from route output")

// DiscoverGatewayIPv4 finds the IPv4 default gateway for macOS.
func DiscoverGatewayIPv4() (net.IP, error) {
	return discoverGateway("route -n get default | grep 'gateway' | awk '{print $2}'")
}

// DiscoverGatewayIPv6 finds the IPv6 default gateway for macOS.
func DiscoverGatewayIPv6() (net.IP, error) {
	return discoverGateway("route -6 -n get default | grep 'gateway' | awk '{print $2}'")
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
