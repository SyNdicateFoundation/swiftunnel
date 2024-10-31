//go:build windows && arm64

package wintun

import _ "embed"

//go:embed bin/arm64/wintun.dll
var wintunDLLData []byte
