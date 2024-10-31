package swiftypes

import (
	"fmt"
	"net"
)

type DNSConfig struct {
	Domain     string
	DnsServers []net.IP
}

var NilGUID = GUID{}
var NilLUID = LUID{}
var NilDNSConfig = &DNSConfig{}

type AdapterType int

const (
	AdapterTypeTUN AdapterType = iota
	AdapterTypeTAP
)

func (g DNSConfig) String() string {
	servers := make([]string, len(g.DnsServers))
	for i, server := range g.DnsServers {
		servers[i] = server.String()
	}
	return fmt.Sprintf("DNSConfig{Domain: %q, DnsServers: %v}", g.Domain, servers)
}

func (l LUID) ToUint64() uint64 {
	return uint64(l.HighPart)<<32 + uint64(l.LowPart)
}

func (l LUID) String() string {
	return fmt.Sprintf("LUID{LowPart: 0x%x, HighPart: 0x%x}", l.LowPart, l.HighPart)
}

func (g GUID) String() string {
	return fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X", g.Data1, g.Data2, g.Data3, g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func NewLUID(luid uint64) LUID {
	return LUID{
		LowPart:  uint32(luid),
		HighPart: int32(luid >> 32),
	}
}
