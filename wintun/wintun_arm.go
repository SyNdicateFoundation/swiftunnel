//go:build windows && arm

package wintun

import _ "embed"

//go:embed bin/arm/wintun.dll
var wintunDLLData []byte
