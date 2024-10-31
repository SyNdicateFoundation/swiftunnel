package swifttunnel

import (
	"github.com/XenonCommunity/swifttunnel/swiftypes"
	"net"
)

type Config struct {
	AdapterName string
	AdapterGUID swiftypes.GUID
	AdapterType swiftypes.AdapterType
	MTU         uint32
	DNSConfig   *swiftypes.DNSConfig
	UnicastIP   net.IPNet
}

var NilConfig = Config{}
