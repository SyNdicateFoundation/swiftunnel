package swiftypes

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type DNSConfig struct {
	Domain     string
	DnsServers []net.IP
}

var NilGUID = GUID{}
var NilLUID = LUID{}

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

func ParseGUID(guid string) (result GUID, err error) {
	parts := strings.Split(guid, "-")
	if len(parts) != 5 {
		return result, fmt.Errorf("invalid GUID format: %q", guid)
	}

	var (
		data1, _ = strconv.ParseUint(parts[0], 16, 32)
		data2, _ = strconv.ParseUint(parts[1], 16, 16)
		data3, _ = strconv.ParseUint(parts[2], 16, 16)
		data4    = make([]byte, 8)
	)

	for i, b := range []string{parts[3], parts[4]} {
		v, _ := strconv.ParseUint(b, 16, 8)
		data4[i*2] = byte(v >> 8)
		data4[i*2+1] = byte(v)
	}

	result = GUID{
		Data1: uint32(data1),
		Data2: uint16(data2),
		Data3: uint16(data3),
		Data4: [8]byte(data4),
	}

	return result, nil
}
