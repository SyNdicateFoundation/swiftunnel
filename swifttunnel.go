package swifttunnel

import (
	"github.com/XenonCommunity/swifttunnel/swiftypes"
	"net"
	"os"
)

type SwiftAdapter interface {
	Write(buf []byte) (int, error)
	Read(buf []byte) (int, error)
	Close() error

	File() *os.File

	GetAdapterName() (string, error)
	GetAdapterIndex() (uint32, error)

	SetMTU(mtu uint32) error
	SetUnicastIpAddressEntry(entry *net.IPNet) error
	SetDNS(config *swiftypes.DNSConfig) error
}
