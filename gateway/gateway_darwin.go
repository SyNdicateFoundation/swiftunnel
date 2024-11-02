//go:build darwin

package gateway

import (
	"errors"
	"net"
	"os/exec"
	"strings"
)

var errCantParse = errors.New("unable to parse route output")

func DiscoverGatewayIPv4() (net.IP, error) {
	cmd := exec.Command("sh", "-c", "route -n get default | grep 'gateway' | awk '{print $2}'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	ipStr := strings.TrimSpace(string(output))
	ipv4 := net.ParseIP(ipStr)
	if ipv4 == nil {
		return nil, errCantParse
	}
	return ipv4, nil
}

func DiscoverGatewayIPv6() (net.IP, error) {
	cmd := exec.Command("sh", "-c", "route -6 -n get default | grep 'gateway' | awk '{print $2}'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	ipStr := strings.TrimSpace(string(output))
	ipv6 := net.ParseIP(ipStr)
	if ipv6 == nil {
		return nil, errCantParse
	}
	return ipv6, nil
}
