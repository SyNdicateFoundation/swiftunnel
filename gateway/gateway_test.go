package gateway

import "testing"

func TestDiscoverGatewayIPv4(t *testing.T) {
	ip, err := DiscoverGatewayIPv4()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("IPV4: %v", ip)
}

func TestDiscoverGatewayIPv6(t *testing.T) {
	ip, err := DiscoverGatewayIPv6()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("IPV6: %v", ip)
}
