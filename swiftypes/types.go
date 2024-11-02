package swiftypes

import (
	"errors"
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
	if len(guid) != 36 || guid[8] != '-' || guid[13] != '-' || guid[18] != '-' || guid[23] != '-' {
		return result, errors.New("invalid GUID format")
	}

	guidWithoutHyphens := strings.ReplaceAll(guid, "-", "")
	if len(guidWithoutHyphens) != 32 {
		return result, errors.New("invalid GUID length")
	}

	// Parse each segment and assign to windows.GUID fields
	result.Data1, err = parseHexUint32(guidWithoutHyphens[0:8])
	if err != nil {
		return result, err
	}
	result.Data2, err = parseHexUint16(guidWithoutHyphens[8:12])
	if err != nil {
		return result, err
	}
	result.Data3, err = parseHexUint16(guidWithoutHyphens[12:16])
	if err != nil {
		return result, err
	}

	// Parse Data4, which is 8 bytes
	for i := 0; i < 8; i++ {
		byteValue, err := strconv.ParseUint(guidWithoutHyphens[16+i*2:18+i*2], 16, 8)
		if err != nil {
			return result, err
		}
		result.Data4[i] = byte(byteValue)
	}

	return result, nil
}

// Helper to parse 32-bit hex number
func parseHexUint32(hexStr string) (uint32, error) {
	value, err := strconv.ParseUint(hexStr, 16, 32)
	return uint32(value), err
}

// Helper to parse 16-bit hex number
func parseHexUint16(hexStr string) (uint16, error) {
	value, err := strconv.ParseUint(hexStr, 16, 16)
	return uint16(value), err
}
