//go:build windows && arm

package wintungo

import _ "embed"

//go:embed bin/arm/wintun.dll
var wintunDLLData []byte
