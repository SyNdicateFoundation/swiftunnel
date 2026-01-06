//go:build !windows

package swiftutils

import (
	"io"
	"os/exec"
	"runtime"
)

// FlushUnixDNS executes platform-specific shell commands to purge the DNS cache on macOS and Linux, suppressing all standard and error output.
func FlushDNS() error {
	switch runtime.GOOS {
	case "darwin":
		c1 := exec.Command("dscacheutil", "-flushcache")
		silenceCommand(c1)
		_ = c1.Run()
		c2 := exec.Command("killall", "-HUP", "mDNSResponder")
		silenceCommand(c2)
		return c2.Run()
	case "linux":
		c1 := exec.Command("resolvectl", "flush-caches")
		silenceCommand(c1)
		if err := c1.Run(); err == nil {
			return nil
		}
		c2 := exec.Command("systemd-resolve", "--flush-caches")
		silenceCommand(c2)
		if err := c2.Run(); err == nil {
			return nil
		}
		c3 := exec.Command("nscd", "-i", "hosts")
		silenceCommand(c3)
		return c3.Run()
	default:
		return nil
	}
}

func silenceCommand(cmd *exec.Cmd) {
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
}
