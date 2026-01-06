//go:build linux

package swiftconfig

import (
	"errors"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"net"
)

// Permissions defines user and group ownership for the Linux tunnel device.
type Permissions struct {
	Owner uint
	Group uint
}

// NewPermissions creates a new ownership configuration.
func NewPermissions(owner, group uint) *Permissions {
	return &Permissions{owner, group}
}

// Config holds configuration parameters for a Linux tunnel interface.
type Config struct {
	AdapterName string
	AdapterType swiftypes.AdapterType

	MTU           int
	UnicastConfig *swiftypes.UnicastConfig

	MultiQueue  bool
	Permissions *Permissions
	Persist     bool
}

// New initializes a Config struct with default Linux values and options.
func New(opts ...Option) (*Config, error) {
	cfg := &Config{
		AdapterName: "Swiftunnel VPN",
		AdapterType: swiftypes.AdapterTypeTUN,
		MTU:         1500,
		MultiQueue:  false,
		Persist:     true,
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

// WithAdapterName sets the Linux interface name.
func WithAdapterName(name string) Option {
	return func(c *Config) error {
		if name == "" {
			return errors.New("AdapterName cannot be empty")
		}

		c.AdapterName = name

		return nil
	}
}

// WithMultiQueue toggles support for multiple packet queues.
func WithMultiQueue(multiqueue bool) Option {
	return func(c *Config) error {
		c.MultiQueue = multiqueue
		return nil
	}
}

// WithPersist sets whether the interface remains after the application exits.
func WithPersist(persist bool) Option {
	return func(c *Config) error {
		c.Persist = persist
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

// WithPermissions sets the UID and GID for the interface.
func WithPermissions(permissions *Permissions) Option {
	return func(c *Config) error {
		c.Permissions = permissions
		return nil
	}
}

// WithUnicastIP parses and sets the primary IP/Subnet.
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

// WithMTU sets the interface MTU.
func WithMTU(mtu int) Option {
	return func(c *Config) error {
		if mtu < 576 || mtu > 65535 {
			return errors.New("MTU must be between 576 and 65535")
		}

		c.MTU = mtu
		return nil
	}
}
