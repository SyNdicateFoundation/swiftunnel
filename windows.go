//go:build windows

package swiftunnel

import (
	"errors"
	"github.com/XenonCommunity/swiftunnel/openvpn"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"github.com/XenonCommunity/swiftunnel/wintun"
	"os"
)

var (
	ErrCannotFindAdapter = errors.New("cannot find adapter")
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

type SwiftInterface struct {
	service swiftService
}

func (a *SwiftInterface) Write(buf []byte) (int, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return a.service.Write(buf)
}

func (a *SwiftInterface) Read(buf []byte) (int, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return a.service.Read(buf)
}

func (a *SwiftInterface) Close() error {
	if a.service == nil {
		return ErrCannotFindAdapter
	}

	return a.service.Close()
}

func (a *SwiftInterface) GetFD() *os.File {
	if a.service == nil {
		return nil
	}

	return a.service.GetFD()
}

func NewSwiftInterface(config Config) (*SwiftInterface, error) {
	adapter := &SwiftInterface{}
	var err error

	switch config.DriverType {
	case DriverTypeWintun:
		if config.AdapterType == swiftypes.AdapterTypeTAP {
			return nil, errors.New("TAP adapter not supported on wintun")
		}
		adap, err := wintun.NewWintunAdapterWithGUID(config.AdapterName, config.AdapterTypeName, config.AdapterGUID)
		if err != nil {
			return nil, err
		}

		adapter.service, err = adap.StartSession(0x800000)

		if config.UnicastConfig != nil {
			if err = adapter.SetUnicastIpAddressEntry(config.UnicastConfig); err != nil {
				return nil, err
			}
		}
	case DriverTypeOpenVPN:
		if config.UnicastConfig == nil {
			return nil, errors.New("unicast IP not specified")
		}

		adapter.service, err = openvpn.NewOpenVPNAdapter(
			config.AdapterGUID,
			config.AdapterName,
			config.UnicastConfig.IP,
			config.UnicastConfig.IPNet,
			config.AdapterType == swiftypes.AdapterTypeTAP,
		)
	default:
		return nil, errors.New("unknown adapter type")
	}

	if err != nil || adapter == nil {
		return nil, err
	}

	if config.MTU != 0 {
		if err = adapter.SetMTU(config.MTU); err != nil {
			return nil, err
		}
	}

	if config.DNSConfig != nil {
		if err = adapter.SetDNS(config.DNSConfig); err != nil {
			return nil, err
		}
	}

	return adapter, nil
}
