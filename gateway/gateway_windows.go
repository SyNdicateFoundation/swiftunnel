//go:build windows

package gateway

import (
	"net"
	"os/exec"
	"strings"
	"syscall"
)

// DiscoverGatewayIPv4 finds the IPv4 gateway using Windows API.
func DiscoverGatewayIPv4() (net.IP, error) {
	routeCmd := exec.Command("route", "print", "0.0.0.0")
	routeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseWindowsGatewayIPv4(output)
}

// DiscoverGatewayIPv6 finds the IPv6 gateway using Windows API.
func DiscoverGatewayIPv6() (net.IP, error) {
	routeCmd := exec.Command("route", "print", "-6", "::/0")
	routeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseWindowsGatewayIPv6(output)
}

// discoverGateway retrieves the gateway address for the specified address family.
type windowsRouteStructIPv4 struct {
	Destination string
	Netmask     string
	Gateway     string
	Interface   string
	Metric      string
}

type windowsRouteStructIPv6 struct {
	If          string
	Metric      string
	Destination string
	Gateway     string
}

func parseToWindowsRouteStructIPv4(output []byte) (windowsRouteStructIPv4, error) {
	// Windows route output format is always like this:
	// ===========================================================================
	// Interface List
	// 8 ...00 12 3f a7 17 ba ...... Intel(R) PRO/100 VE Network Connection
	// 1 ........................... Software Loopback Interface 1
	// ===========================================================================
	// IPv4 Route Table
	// ===========================================================================
	// Active Routes:
	// Network Destination        Netmask          Gateway       Interface  Metric
	//           0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     20
	// ===========================================================================
	//
	// Windows commands are localized, so we can't just look for "Active Routes:" string
	// I'm trying to pick the active route,
	// then jump 2 lines and get the row
	// Not using regex because output is quite standard from Windows XP to 8 (NEEDS TESTING)
	lines := strings.Split(string(output), "\n")
	sep := 0
	for idx, line := range lines {
		if sep == 3 {
			// We just entered the 2nd section containing "Active Routes:"
			if len(lines) <= idx+2 {
				return windowsRouteStructIPv4{}, ErrNoGateway
			}

			fields := strings.Fields(lines[idx+2])
			if len(fields) < 5 {
				return windowsRouteStructIPv4{}, ErrCantParse
			}

			return windowsRouteStructIPv4{
				Destination: fields[0],
				Netmask:     fields[1],
				Gateway:     fields[2],
				Interface:   fields[3],
				Metric:      fields[4],
			}, nil
		}
		if strings.HasPrefix(line, "=======") {
			sep++
			continue
		}
	}
	return windowsRouteStructIPv4{}, ErrNoGateway
}

func parseToWindowsRouteStructIPv6(output []byte) (windowsRouteStructIPv6, error) {

	lines := strings.Split(string(output), "\n")
	sep := 0
	for idx, line := range lines {
		if sep == 3 {
			// We just entered the 2nd section containing "Active Routes:"
			if len(lines) <= idx+2 {
				return windowsRouteStructIPv6{}, ErrNoGateway
			}

			fields := strings.Fields(lines[idx+2])
			if len(fields) < 4 {
				return windowsRouteStructIPv6{}, ErrCantParse
			}

			return windowsRouteStructIPv6{
				If:          fields[0],
				Metric:      fields[1],
				Destination: fields[2],
				Gateway:     fields[3],
			}, nil
		}
		if strings.HasPrefix(line, "=======") {
			sep++
			continue
		}
	}
	return windowsRouteStructIPv6{}, ErrNoGateway
}

func parseWindowsGatewayIPv4(output []byte) (net.IP, error) {
	parsedOutput, err := parseToWindowsRouteStructIPv4(output)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(parsedOutput.Gateway)
	if ip == nil {
		return nil, ErrCantParse
	}
	return ip, nil
}

func parseWindowsGatewayIPv6(output []byte) (net.IP, error) {
	parsedOutput, err := parseToWindowsRouteStructIPv6(output)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(parsedOutput.Gateway)
	if ip == nil {
		return nil, ErrCantParse
	}
	return ip, nil
}
