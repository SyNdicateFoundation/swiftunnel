//go:build windows

package swiftconfig

import (
	"errors"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"golang.org/x/sys/windows"
	"net"
)

// DriverType specifies whether to use Wintun or TAP-Windows.
type DriverType int

const (
	DriverTypeWintun DriverType = iota
	DriverTypeOpenVPN
)

// Config holds configuration parameters for a Windows tunnel interface.
type Config struct {
	AdapterName     string
	AdapterTypeName string
	AdapterGUID     swiftypes.GUID
	AdapterType     swiftypes.AdapterType
	DriverType      DriverType

	RingBuffer    uint32
	MTU           int
	DNSConfig     *swiftypes.DNSConfig
	UnicastConfig *swiftypes.UnicastConfig
}

// New initializes a Config struct with default Windows values.
func New(opts ...Option) (*Config, error) {
	cfg := &Config{
		AdapterName:     "Swiftunnel VPN",
		AdapterTypeName: "Swiftunnel",
		AdapterType:     swiftypes.AdapterTypeTUN,
		DriverType:      DriverTypeWintun,
		MTU:             1500,
		RingBuffer:      0x800000,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	if cfg.UnicastConfig == nil {
		return nil, errors.New("unicast configuration (IP/Net) is required")
	}

	return cfg, nil
}

// WithAdapterName sets the friendly name of the Windows adapter.
func WithAdapterName(name string) Option {
	return func(c *Config) error {
		if name == "" {
			return errors.New("AdapterName cannot be empty")
		}

		c.AdapterName = name
		return nil
	}
}

// WithAdapterTypeName sets the adapter class name (e.g., "Wintun").
func WithAdapterTypeName(name string) Option {
	return func(c *Config) error {
		if name == "" {
			return errors.New("adapter type name cannot be empty")
		}

		c.AdapterTypeName = name
		return nil
	}
}

// WithAdapterType specifies TUN or TAP.
func WithAdapterType(adapterType swiftypes.AdapterType) Option {
	return func(c *Config) error {
		c.AdapterType = adapterType
		return nil
	}
}

// WithDriverType selects Wintun or OpenVPN driver.
func WithDriverType(driverType DriverType) Option {
	return func(c *Config) error {
		c.DriverType = driverType
		return nil
	}
}

// WithDNSConfig applies DNS settings to the adapter.
func WithDNSConfig(dnsConfig *swiftypes.DNSConfig) Option {
	return func(c *Config) error {
		c.DNSConfig = dnsConfig
		return nil
	}
}

// WithUnicastIP sets the CIDR address.
func WithUnicastIP(ipStr string) Option {
	return func(c *Config) error {
		ip, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return err
		}

		c.UnicastConfig = &swiftypes.UnicastConfig{
			IP:       ip,
			DadState: windows.IpDadStatePreferred,
			IPNet:    ipNet,
		}

		return nil
	}
}

// WithUnicastConfig parses a CIDR string into an IP and Network mask.
func WithUnicastConfig(unicast *swiftypes.UnicastConfig) Option {
	return func(c *Config) error {
		c.UnicastConfig = unicast
		return nil
	}
}

// WithMTU sets the adapter MTU.
func WithMTU(mtu int) Option {
	return func(c *Config) error {
		if mtu < 576 || mtu > 65535 {
			return errors.New("MTU must be between 576 and 65535")
		}

		c.MTU = mtu
		return nil
	}
}

// WithRingBuffer sets the adapter RingBuffer.
func WithRingBuffer(ringBuffer uint32) Option {
	return func(c *Config) error {
		if ringBuffer < 1024 {
			return errors.New("MTU must be between 576 and 20MB")
		}

		c.RingBuffer = ringBuffer
		return nil
	}
}

// WithGUIDStr assigns a specific GUID to the adapter.
func WithGUIDStr(guidStr string) Option {
	return func(c *Config) error {
		g, err := swiftypes.ParseGUID(guidStr)
		if err != nil {
			return err
		}

		c.AdapterGUID = g

		return nil
	}
}

// WithGUID assigns a specific GUID to the adapter.
func WithGUID(guid swiftypes.GUID) Option {
	return func(c *Config) error {
		c.AdapterGUID = guid
		return nil
	}
}
