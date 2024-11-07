//go:build windows

package gateway

import (
	"errors"
	"golang.org/x/sys/windows"
	"net"
	"unsafe"
)

// DiscoverGatewayIPv4 finds the IPv4 gateway using Windows API.
func DiscoverGatewayIPv4() (net.IP, error) {
	return discoverGateway(windows.AF_INET)
}

// DiscoverGatewayIPv6 finds the IPv6 gateway using Windows API.
func DiscoverGatewayIPv6() (net.IP, error) {
	return discoverGateway(windows.AF_INET6)
}

// discoverGateway retrieves the gateway address for the specified address family.
func discoverGateway(family uint16) (net.IP, error) {
	var buffer []byte
	size := uint32(15000)

	for {
		buffer = make([]byte, size)
		err := windows.GetAdaptersAddresses(uint32(family), windows.GAA_FLAG_INCLUDE_GATEWAYS, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0])), &size)
		if err == nil {
			break
		}

		// Check for buffer overflow error and resize if necessary.
		if !errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) || size <= uint32(len(buffer)) {
			return nil, err
		}
	}

	var adapters []*windows.IpAdapterAddresses
	for adapter := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0])); adapter != nil; adapter = adapter.Next {
		adapters = append(adapters, adapter)
	}

	// Iterate over adapters to find the first gateway address.
	for _, adapter := range adapters {
		if adapter.FirstGatewayAddress != nil {
			return adapter.FirstGatewayAddress.Address.IP(), nil
		}
	}

	return nil, ErrNoGateway
}
