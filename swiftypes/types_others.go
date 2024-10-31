//go:build !windows

package swiftypes

import (
	"syscall"
)

type GUID syscall.GUID

type LUID struct {
	LowPart  uint32
	HighPart int32
}
