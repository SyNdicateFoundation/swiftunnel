//go:build 386 || arm

package wintungo

const (
	ifMaxStringSize        = 256
	ifMaxPhysAddressLength = 32
	ScopeLevelCount        = 16
)

type nibUnicastIPAddressRow struct {
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
	DadState           nlDadState
	ScopeID            uint32
	CreationTimeStamp  int64
}

type mibIfrow struct {
	WszName           [maxInterfaceNameLen]uint16 // Interface name
	DwIndex           uint32                      // Interface index
	DwType            uint32                      // Interface type
	DwMtu             uint32                      // MTU size
	DwSpeed           uint32                      // Speed (in bits per second)
	DwPhysAddrLen     uint32                      // Physical address length
	BPhysAddr         [maxlenPhysaddr]byte        // Physical address (MAC address)
	DwAdminStatus     uint32                      // Administrative status
	DwOperStatus      uint32                      // Operational status
	DwLastChange      uint32                      // Last change timestamp
	DwInOctets        uint32                      // Incoming octets
	DwInUcastPkts     uint32                      // Incoming unicast packets
	DwInNUcastPkts    uint32                      // Incoming non-unicast packets
	DwInDiscards      uint32                      // Incoming discards
	DwInErrors        uint32                      // Incoming errors
	DwInUnknownProtos uint32                      // Unknown protocols
	DwOutOctets       uint32                      // Outgoing octets
	DwOutUcastPkts    uint32                      // Outgoing unicast packets
	DwOutNUcastPkts   uint32                      // Outgoing non-unicast packets
	DwOutDiscards     uint32                      // Outgoing discards
	DwOutErrors       uint32                      // Outgoing errors
	DwOutQLen         uint32                      // Output queue length
	DwDescrLen        uint32                      // Description length
	BDescr            [maxlenIfdescr]byte         // Description
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
