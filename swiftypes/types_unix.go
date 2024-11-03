//go:build !windows

package swiftypes

import (
	"net"
)

const (
	InterfaceUp InterfaceStatus = iota
	InterfaceDown
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type LUID struct {
	LowPart  uint32
	HighPart int32
}

type UnicastConfig struct {
	IPNet       *net.IPNet
	Gateway, IP net.IP
}
