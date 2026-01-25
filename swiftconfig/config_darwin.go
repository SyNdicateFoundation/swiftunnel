//go:build darwin

package swiftconfig

import (
	"errors"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"net"
)

// DriverType specifies the underlying macOS tunnel implementation.
type DriverType int

const (
	DriverTypeTunTapOSX DriverType = iota
	DriverTypeSystem
)

// Config holds configuration parameters for a macOS tunnel interface.
type Config struct {
	AdapterName string
	AdapterType swiftypes.AdapterType
	DriverType  DriverType

	MTU           int
	UnicastConfig *swiftypes.UnicastConfig
}

// New initializes a Config struct with default values and functional options.
func New(opts ...Option) (*Config, error) {
	cfg := &Config{
		AdapterName: "swiftunnelvpn",
		AdapterType: swiftypes.AdapterTypeTUN,
		MTU:         1500,
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

// WithAdapterName sets the friendly name of the adapter.
func WithAdapterName(name string) Option {
	return func(c *Config) error {
		if name == "" {
			return errors.New("AdapterName cannot be empty")
		}

		c.AdapterName = name

		return nil
	}
}

// WithDriverType selects between System or TunTapOSX drivers.
func WithDriverType(driverType DriverType) Option {
	return func(c *Config) error {
		c.DriverType = driverType
		return nil
	}
}

// WithAdapterType specifies TUN or TAP behavior.
func WithAdapterType(adapterType swiftypes.AdapterType) Option {
	return func(c *Config) error {
		c.AdapterType = adapterType
		return nil
	}
}

// WithUnicastIP parses a CIDR string into an IP and Network mask.
func WithUnicastIP(ipStr string) Option {
	return func(c *Config) error {
		ip, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return err
		}

		c.UnicastConfig = &swiftypes.UnicastConfig{
			IP:    ip,
			IPNet: ipNet,
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

// WithMTU sets the Maximum Transmission Unit.
func WithMTU(mtu int) Option {
	return func(c *Config) error {
		if mtu < 576 || mtu > 65535 {
			return errors.New("MTU must be between 576 and 65535")
		}

		c.MTU = mtu
		return nil
	}
}
