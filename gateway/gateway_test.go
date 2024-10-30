package gateway

import "testing"

func TestDiscoverGatewayIPv4(t *testing.T) {
	ip, err := DiscoverGatewayIPv4()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("IPv4: %v", ip)
}

func TestDiscoverGatewayIPv6(t *testing.T) {
	ip, err := DiscoverGatewayIPv6()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("IPv6: %v", ip)
}
