//go:build !windows

package swiftypes

import (
	"net"
)

const (
	InterfaceUp InterfaceStatus = iota
	InterfaceDown
)

// GUID is a Unix-compatible representation of a Globally Unique Identifier.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// LUID is a Unix-compatible representation of a Locally Unique Identifier.
type LUID struct {
	LowPart  uint32
	HighPart int32
}

// UnicastConfig holds IP addressing information for Unix systems.
type UnicastConfig struct {
	IPNet       *net.IPNet
	Gateway, IP net.IP
}
