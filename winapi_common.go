//go:build windows

package Swiftunnel

type DadState uint32

const (
	IpDadStateInvalid DadState = iota
	IpDadStateTentative
	IpDadStateDuplicate
	IpDadStateDeprecated
	IpDadStatePreferred
)

type sockaddrInet struct {
	Family uint16
	Data   [26]byte
}
