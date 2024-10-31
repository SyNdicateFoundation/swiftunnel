//go:build darwin

package gateway

import (
	"net"
	"os/exec"
)

func DiscoverGatewayIPv4() (ip net.IP, err error) {
	ipstr, err := exec.Command("sh", "-c", "route -n get default | grep 'gateway' | awk 'NR==1{print $2}'").CombinedOutput()
	if err != nil {
		return nil, err
	}

	ipv4 := net.ParseIP(ipstr)
	if ipv4 == nil {
		return nil, errCantParse
	}
	return ipv4, nil
}

func DiscoverGatewayIPv6() (ip net.IP, err error) {
	ipstr, err := exec.Command("sh", "-c", "route -6 -n get default | grep 'gateway' | awk 'NR==1{print $2}'").CombinedOutput()
	if err != nil {
		return nil, err
	}

	ipv6 := net.ParseIP(ipstr)
	if ipv6 == nil {
		return nil, errCantParse
	}
	return ipv6, nil
}
