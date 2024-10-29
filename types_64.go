//go:build amd64 || arm64

package wintungo

const (
	maxInterfaceNameLen = 256
	maxlenPhysaddr      = 8
	maxlenIfdescr       = 256
)

type nibUnicastIPAddressRow struct {
	Address            sockaddrInet
	InterfaceLUID      uint64
	InterfaceIndex     uint32
	PrefixOrigin       uint32
	SuffixOrigin       uint32
	ValidLifetime      uint32
	PreferredLifetime  uint32
	OnLinkPrefixLength uint8
	SkipAsSource       bool
	DadState           nlDadState
	ScopeID            uint32
	CreationTimeStamp  int64
}

type mibIfrow struct {
	WszName           [maxInterfaceNameLen]uint16 // WCHAR is equivalent to uint16 in Go
	DwIndex           uint32                      // IF_INDEX (equivalent to DWORD in Go)
	DwType            uint32                      // IFTYPE (also DWORD)
	DwMtu             uint32
	DwSpeed           uint32
	DwPhysAddrLen     uint32
	BPhysAddr         [maxlenPhysaddr]byte
	DwAdminStatus     uint32
	DwOperStatus      uint32 // INTERNAL_IF_OPER_STATUS (equivalent to DWORD in Go)
	DwLastChange      uint32
	DwInOctets        uint32
	DwInUcastPkts     uint32
	DwInNUcastPkts    uint32
	DwInDiscards      uint32
	DwInErrors        uint32
	DwInUnknownProtos uint32
	DwOutOctets       uint32
	DwOutUcastPkts    uint32
	DwOutNUcastPkts   uint32
	DwOutDiscards     uint32
	DwOutErrors       uint32
	DwOutQLen         uint32
	DwDescrLen        uint32
	BDescr            [maxlenIfdescr]byte
}

type dnsInterfaceSettings struct {
	Version             uint32
	Flags               uint64
	Domain              *uint16
	NameServer          *uint16
	SearchList          *uint16
	RegistrationEnabled uint32
	RegisterAdapterName uint32
	EnableLLMNR         uint32
	QueryAdapterName    uint32
	ProfileNameServer   *uint16
}
