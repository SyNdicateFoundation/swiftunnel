//go:build windows

package swiftypes

import (
	"golang.org/x/sys/windows"
)

type GUID windows.GUID
type LUID windows.LUID
