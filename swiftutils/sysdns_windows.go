//go:build windows

package swiftutils

import (
	"errors"
	"golang.org/x/sys/windows"
)

var (
	dnsapi                = windows.NewLazySystemDLL("dnsapi.dll")
	dnsFlushResolverCache = dnsapi.NewProc("DnsFlushResolverCache")
)

// FlushDNS executes platform-specific shell commands to purge the DNS cache on macOS and Linux, suppressing all standard and error output.
func FlushDNS() error {
	ret, _, _ := dnsFlushResolverCache.Call()
	if ret == 0 {
		return errors.New("failed to flush dns cache via native api")
	}

	return nil
}
