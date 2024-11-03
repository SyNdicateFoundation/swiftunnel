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
	GetAdapterIndex() (uint32, error)
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

func (a *SwiftInterface) GetAdapterName() (string, error) {
	if a.service == nil {
		return "", ErrCannotFindAdapter
	}

	return a.service.GetAdapterName()
}

func (a *SwiftInterface) GetAdapterIndex() (uint32, error) {
	if a.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return a.service.GetAdapterIndex()
}

func (a *SwiftInterface) SetMTU(mtu int) error {
	adapterIndex, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	return setMTU(adapterIndex, mtu)
}

func (a *SwiftInterface) SetUnicastIpAddressEntry(config *swiftypes.UnicastConfig) error {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return err
	}

	return setUnicastIpAddressEntry(luid, config)
}

func (a *SwiftInterface) SetDNS(config *swiftypes.DNSConfig) error {
	guid, err := a.GetAdapterGUID()
	if err != nil {
		return err
	}

	return setDNS(guid, config)
}

func (a *SwiftInterface) GetAdapterLUID() (swiftypes.LUID, error) {
	if a.service == nil {
		return swiftypes.NilLUID, ErrCannotFindAdapter
	}

	return a.service.GetAdapterLUID()
}

func (a *SwiftInterface) GetAdapterGUID() (swiftypes.GUID, error) {
	if a.service == nil {
		return swiftypes.NilGUID, ErrCannotFindAdapter
	}

	return a.service.GetAdapterGUID()
}

func (a *SwiftInterface) SetStatus(status swiftypes.InterfaceStatus) error {
	index, err := a.GetAdapterIndex()
	if err != nil {
		return err
	}

	return setInterfaceStatus(index, status)
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
