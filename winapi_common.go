//go:build windows

package swiftunnel

type dnsSettingFlags uint64

const (
	dnsSettingIpv6                dnsSettingFlags = 0x0001
	dnsSettingNameserver          dnsSettingFlags = 0x0002
	dnsSettingSearchlist          dnsSettingFlags = 0x0004
	dnsSettingRegistrationEnabled dnsSettingFlags = 0x0008
	dnsSettingDomain              dnsSettingFlags = 0x0020
	dnsSettingsEnableLlmnr        dnsSettingFlags = 0x0080
	dnsSettingsQueryAdapterName   dnsSettingFlags = 0x0100
	dnsSettingProfileNameserver   dnsSettingFlags = 0x0200
)

type sockaddrInet struct {
	Family  uint16
	Port    uint16   // Optional: if you need to specify a port (common in socket programming)
	Addr    [16]byte // Buffer for the address; use 16 bytes to accommodate the IPv6 address.
	Padding [6]byte  // Padding to make the structure size consistent with typical socket structures.
}
