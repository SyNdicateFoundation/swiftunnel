//go:build windows

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel/openvpn"
	"github.com/SyNdicateFoundation/swiftunnel/swiftconfig"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"github.com/SyNdicateFoundation/swiftunnel/wintun"
	"os"
)

var (
	ErrCannotFindAdapter = errors.New("cannot find adapter")
	ErrInvalidDriver     = errors.New("unknown driver type")
)

type swiftService interface {
	Write(buf []byte) (int, error)
	Read(buf []byte) (int, error)
	Close() error

	GetFD() *os.File
	GetAdapterName() (string, error)
	GetAdapterIndex() (int, error)
	GetAdapterLUID() (swiftypes.LUID, error)
	GetAdapterGUID() (swiftypes.GUID, error)
}

// SwiftInterface provides a generic interface for Windows network tunnels.
type SwiftInterface struct {
	service swiftService
}

// Write transmits a packet via the underlying Windows service.
func (a *SwiftInterface) Write(buf []byte) (int, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}
	return a.service.Write(buf)
}

// Read receives a packet via the underlying Windows service.
func (a *SwiftInterface) Read(buf []byte) (int, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}
	return a.service.Read(buf)
}

// Close terminates the adapter session and releases driver resources.
func (a *SwiftInterface) Close() error {
	if a.service == nil {
		return nil
	}
	return a.service.Close()
}

// GetFD retrieves the file handle associated with the tunnel session.
func (a *SwiftInterface) GetFD() *os.File {
	if a.service == nil {
		return nil
	}
	return a.service.GetFD()
}

// NewSwiftInterface creates a new adapter using either the Wintun or TAP-Windows driver.
func NewSwiftInterface(config *swiftconfig.Config) (*SwiftInterface, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	adapter := &SwiftInterface{}
	var err error

	switch config.DriverType {
	case swiftconfig.DriverTypeWintun:
		if config.AdapterType == swiftypes.AdapterTypeTAP {
			return nil, errors.New("TAP adapter not supported on Wintun driver")
		}

		adap, err := wintun.NewWintunAdapterWithGUID(config.AdapterName, config.AdapterTypeName, config.AdapterGUID)
		if err != nil {
			return nil, fmt.Errorf("failed to create wintun adapter: %w", err)
		}

		session, err := adap.StartSession(config.RingBuffer)
		if err != nil {
			_ = adap.Close()
			return nil, fmt.Errorf("failed to start wintun session: %w", err)
		}
		adapter.service = session

	case swiftconfig.DriverTypeOpenVPN:
		if config.UnicastConfig == nil {
			return nil, errors.New("unicast IP must be specified for TAP driver")
		}

		adapter.service, err = openvpn.NewOpenVPNAdapter(
			config.AdapterGUID,
			config.AdapterName,
			config.UnicastConfig.IP,
			config.UnicastConfig.IPNet,
			config.AdapterType == swiftypes.AdapterTypeTAP,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create openvpn adapter: %w", err)
		}

	default:
		return nil, ErrInvalidDriver
	}

	if err := configureAdapter(adapter, config); err != nil {
		_ = adapter.Close()
		return nil, err
	}

	return adapter, nil
}

// configureAdapter applies IP, MTU, and DNS settings to a newly created interface.
func configureAdapter(adapter *SwiftInterface, config *swiftconfig.Config) error {
	if config.UnicastConfig != nil {
		if err := adapter.SetUnicastIpAddressEntry(config.UnicastConfig); err != nil {
			return fmt.Errorf("failed to set IP address: %w", err)
		}
	}

	if config.MTU > 0 {
		if err := adapter.SetMTU(config.MTU); err != nil {
			return fmt.Errorf("failed to set MTU: %w", err)
		}
	}

	if config.DNSConfig != nil {
		if err := adapter.SetDNS(config.DNSConfig); err != nil {
			return fmt.Errorf("failed to set DNS: %w", err)
		}
	}

	return nil
}
