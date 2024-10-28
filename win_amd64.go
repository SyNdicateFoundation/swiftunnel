//go:build windows && amd64

package wintun

import _ "embed"

//go:embed bin/amd64/wintun.dll
var wintunDLLData []byte
