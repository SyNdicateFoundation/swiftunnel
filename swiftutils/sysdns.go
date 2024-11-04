package swiftutils

import (
	"os/exec"
	"runtime"
)

func FlushDNSCache() error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("ipconfig", "/flushdns").Run()
	case "linux":
		return exec.Command("sh", "-c", "sudo systemd-resolve --flush-caches").Run()
	case "darwin":
		return exec.Command("sudo", "dscacheutil", "-flushcache").Run()
	default:
		return nil
	}
}
