//go:build windows

package swiftypes

import (
	"golang.org/x/sys/windows"
	"net"
)

type DadState uint32

const (
	IpDadStateInvalid DadState = iota
	IpDadStateTentative
	IpDadStateDuplicate
	IpDadStateDeprecated
	IpDadStatePreferred
)

const (
	InterfaceUp InterfaceStatus = iota
	InterfaceDown
)

type GUID windows.GUID
type LUID windows.LUID

type UnicastConfig struct {
	IPNet       *net.IPNet
	Gateway, IP net.IP
	DadState    DadState
}
