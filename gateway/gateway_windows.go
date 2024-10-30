//go:build windows

package gateway

import (
	"errors"
	"net"
	"os/exec"
	"strings"
	"syscall"
)

var (
	ErrNoGateway = errors.New("no gateway found")
	ErrCantParse = errors.New("unable to parse route output")
)

func DiscoverGatewayIPv4() (net.IP, error) {
	return discoverGateway("0.0.0.0", parseIPv4RouteEntry)
}

func DiscoverGatewayIPv6() (net.IP, error) {
	return discoverGateway("-6 ::/0", parseIPv6RouteEntry)
}

func discoverGateway(routeArg string, parseFunc func([]byte) (string, error)) (net.IP, error) {
	cmd := exec.Command("route", "print", routeArg)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	gatewayIPStr, err := parseFunc(output)
	if err != nil {
		return nil, err
	}

	gatewayIP := net.ParseIP(gatewayIPStr)
	if gatewayIP == nil {
		return nil, ErrCantParse
	}

	return gatewayIP, nil
}

func parseIPv4RouteEntry(output []byte) (string, error) {
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Active Routes") {
			if len(lines) <= i+2 {
				return "", ErrNoGateway
			}
			fields := strings.Fields(lines[i+2])
			if len(fields) < 5 {
				return "", ErrCantParse
			}
			return fields[2], nil
		}
	}
	return "", ErrNoGateway
}

func parseIPv6RouteEntry(output []byte) (string, error) {
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Active Routes") {
			if len(lines) <= i+2 {
				return "", ErrNoGateway
			}
			fields := strings.Fields(lines[i+2])
			if len(fields) < 4 {
				return "", ErrCantParse
			}
			return fields[3], nil
		}
	}
	return "", ErrNoGateway
}
