//go:build windows && (386 || arm)

package swiftunnel

type dnsInterfaceSettings struct {
	Version             uint32
	Flags               dnsSettingFlags
	_                   uint32
	Domain              *uint16
	NameServer          *uint16
	SearchList          *uint16
	RegistrationEnabled uint32
	RegisterAdapterName uint32
	EnableLLMNR         uint32
	QueryAdapterName    uint32
	ProfileNameServer   *uint16
}
