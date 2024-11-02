//go:build windows && (386 || arm)

package swiftunnel

import "github.com/XenonCommunity/swiftunnel/swiftypes"

type mibUnicastIPAddressRow struct {
	Address            sockaddrInet
	_                  [4]byte
	InterfaceLUID      uint64
	InterfaceIndex     uint32
	PrefixOrigin       uint32
	SuffixOrigin       uint32
	ValidLifetime      uint32
	PreferredLifetime  uint32
	OnLinkPrefixLength uint8
	SkipAsSource       bool
	DadState           swiftypes.DadState
	ScopeID            uint32
	CreationTimeStamp  int64
}

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
