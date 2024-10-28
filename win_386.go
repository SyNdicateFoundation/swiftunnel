//go:build windows && 386

package wintun

import _ "embed"

//go:embed bin/x86/wintun.dll
var wintunDLLData []byte
