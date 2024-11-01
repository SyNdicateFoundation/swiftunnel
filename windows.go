//go:build windows

package swiftunnel

import (
	"errors"
	"github.com/XenonCommunity/swiftunnel/openvpn"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"github.com/XenonCommunity/swiftunnel/wintun"
	"net"
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

func (w *SwiftInterface) Write(buf []byte) (int, error) {
	if w.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return w.service.Write(buf)
}

func (w *SwiftInterface) Read(buf []byte) (int, error) {
	if w.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return w.service.Read(buf)
}

func (w *SwiftInterface) Close() error {
	if w.service == nil {
		return ErrCannotFindAdapter
	}

	return w.service.Close()
}

func (w *SwiftInterface) GetFD() *os.File {
	if w.service == nil {
		return nil
	}

	return w.service.GetFD()
}

func (w *SwiftInterface) GetAdapterName() (string, error) {
	if w.service == nil {
		return "", ErrCannotFindAdapter
	}

	return w.service.GetAdapterName()
}

func (w *SwiftInterface) GetAdapterIndex() (uint32, error) {
	if w.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return w.service.GetAdapterIndex()
}

func (w *SwiftInterface) SetMTU(mtu int) error {
	adapterIndex, err := w.GetAdapterIndex()
	if err != nil {
		return err
	}

	return setMTU(adapterIndex, mtu)
}

func (w *SwiftInterface) SetUnicastIpAddressEntry(entry *net.IPNet) error {
	luid, err := w.GetAdapterLUID()
	if err != nil {
		return err
	}

	return setUnicastIpAddressEntry(luid, entry, IpDadStatePreferred)
}

func (w *SwiftInterface) SetDNS(config *swiftypes.DNSConfig) error {
	guid, err := w.GetAdapterGUID()
	if err != nil {
		return err
	}

	return setDNS(guid, config)
}

func (w *SwiftInterface) GetAdapterLUID() (swiftypes.LUID, error) {
	if w.service == nil {
		return swiftypes.NilLUID, ErrCannotFindAdapter
	}

	return w.service.GetAdapterLUID()
}

func (w *SwiftInterface) GetAdapterGUID() (swiftypes.GUID, error) {
	if w.service == nil {
		return swiftypes.NilGUID, ErrCannotFindAdapter
	}

	return w.service.GetAdapterGUID()
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

		if config.UnicastIP != nil {
			if err = adapter.SetUnicastIpAddressEntry(config.UnicastIP); err != nil {
				return nil, err
			}
		}
	case DriverTypeOpenVPN:
		if config.UnicastIP == nil {
			return nil, errors.New("unicast IP not specified")
		}

		adapter.service, err = openvpn.NewOpenVPNAdapter(
			config.AdapterGUID,
			config.AdapterName,
			config.UnicastIP.IP,
			config.UnicastIP,
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
