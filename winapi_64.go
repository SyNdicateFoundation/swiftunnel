//go:build windows && (amd64 || arm64)

package swiftunnel

type dnsInterfaceSettings struct {
	Version             uint32
	Flags               dnsSettingFlags
	Domain              *uint16
	NameServer          *uint16
	SearchList          *uint16
	RegistrationEnabled uint32
	RegisterAdapterName uint32
	EnableLLMNR         uint32
	QueryAdapterName    uint32
	ProfileNameServer   *uint16
}
