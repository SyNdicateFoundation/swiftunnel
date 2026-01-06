package gateway

import "testing"

// TestDiscoverGatewayIPv4 validates discovery of the IPv4 default gateway.
func TestDiscoverGatewayIPv4(t *testing.T) {
	ip, err := DiscoverGatewayIPv4()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("IPv4: %v", ip)
}

// TestDiscoverGatewayIPv6 validates discovery of the IPv6 default gateway.
func TestDiscoverGatewayIPv6(t *testing.T) {
	ip, err := DiscoverGatewayIPv6()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("IPv6: %v", ip)
}
