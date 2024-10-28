//go:build windows && arm64

package wintungo

import _ "embed"

//go:embed bin/arm64/wintun.dll
var wintunDLLData []byte
