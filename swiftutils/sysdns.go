package swiftutils

import (
	"os/exec"
)

func FlushDNSCache() error {
	// Flush DNS cache
	cmd := exec.Command("ipconfig", "/flushdns")
	return cmd.Run()
}
