package wintungo

import "golang.org/x/sys/windows"

// nlDadState represents the duplicate address detection state.
type nlDadState uint32

const (
	IpDadStateInvalid    nlDadState = iota // 0: The DAD state is invalid.
	IpDadStateTentative                    // 1: The DAD state is tentative.
	IpDadStateDuplicate                    // 2: A duplicate IP address has been detected.
	IpDadStateDeprecated                   // 3: The IP address has been deprecated.
	IpDadStatePreferred                    // 4: The IP address is preferred.
)

type SIFamily uint16

type sockaddrInet struct {
	Family SIFamily
	data   [26]byte
}

func convertLUIDtouint64(luid windows.LUID) uint64 {
	return uint64(luid.HighPart)<<32 + uint64(luid.LowPart)
}
