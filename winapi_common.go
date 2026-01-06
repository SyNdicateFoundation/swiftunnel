//go:build windows

package swiftunnel

type dnsSettingFlags uint64

const (
	dnsSettingIpv6       dnsSettingFlags = 0x0001
	dnsSettingNameserver dnsSettingFlags = 0x0002
	dnsSettingDomain     dnsSettingFlags = 0x0020
)
