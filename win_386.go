//go:build windows && 386

package wintungo

import _ "embed"

//go:embed bin/x86/wintun.dll
var wintunDLLData []byte
