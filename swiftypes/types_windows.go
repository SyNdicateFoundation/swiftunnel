//go:build windows

package swiftypes

import (
	"golang.org/x/sys/windows"
	"net"
)

const (
	InterfaceUp InterfaceStatus = iota
	InterfaceDown
)

// GUID maps to the windows.GUID type.
type GUID windows.GUID

// LUID maps to the windows.LUID type.
type LUID windows.LUID

// UnicastConfig holds IP addressing and Windows-specific DAD state information.
type UnicastConfig struct {
	IPNet       *net.IPNet
	Gateway, IP net.IP
	DadState    uint32
}
