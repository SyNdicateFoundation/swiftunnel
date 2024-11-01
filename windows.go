//go:build windows

package swifttunnel

import (
	"errors"
	"github.com/XenonCommunity/swifttunnel/openvpn"
	"github.com/XenonCommunity/swifttunnel/swiftypes"
	"github.com/XenonCommunity/swifttunnel/wintun"
	"net"
	"os"
)

var (
	ErrCannotFindAdapter = errors.New("cannot find adapter")
)

type SwiftService interface {
	Write(buf []byte) (int, error)
	Read(buf []byte) (int, error)
	Close() error

	File() *os.File

	GetAdapterName() (string, error)
	GetAdapterIndex() (uint32, error)
	GetAdapterLUID() (swiftypes.LUID, error)
	GetAdapterGUID() (swiftypes.GUID, error)
}

type WindowsAdapter struct {
	service SwiftService
}

func (w *WindowsAdapter) Write(buf []byte) (int, error) {
	if w.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return w.service.Write(buf)
}

func (w *WindowsAdapter) Read(buf []byte) (int, error) {
	if w.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return w.service.Read(buf)
}

func (w *WindowsAdapter) Close() error {
	if w.service == nil {
		return ErrCannotFindAdapter
	}

	return w.service.Close()
}

func (w *WindowsAdapter) File() *os.File {
	if w.service == nil {
		return nil
	}

	return w.service.File()
}

func (w *WindowsAdapter) GetAdapterName() (string, error) {
	if w.service == nil {
		return "", ErrCannotFindAdapter
	}

	return w.service.GetAdapterName()
}

func (w *WindowsAdapter) GetAdapterIndex() (uint32, error) {
	if w.service == nil {
		return 0, ErrCannotFindAdapter
	}

	return w.service.GetAdapterIndex()
}

func (w *WindowsAdapter) SetMTU(mtu int) error {
	adapterIndex, err := w.GetAdapterIndex()
	if err != nil {
		return err
	}

	return setMTU(adapterIndex, mtu)
}

func (w *WindowsAdapter) SetUnicastIpAddressEntry(entry *net.IPNet) error {
	luid, err := w.GetAdapterLUID()
	if err != nil {
		return err
	}

	return setUnicastIpAddressEntry(luid, entry, IpDadStatePreferred)
}

func (w *WindowsAdapter) SetDNS(config *swiftypes.DNSConfig) error {
	guid, err := w.GetAdapterGUID()
	if err != nil {
		return err
	}

	return setDNS(guid, config)
}

func (w *WindowsAdapter) GetAdapterLUID() (swiftypes.LUID, error) {
	if w.service == nil {
		return swiftypes.NilLUID, ErrCannotFindAdapter
	}

	return w.service.GetAdapterLUID()
}

func (w *WindowsAdapter) GetAdapterGUID() (swiftypes.GUID, error) {
	if w.service == nil {
		return swiftypes.NilGUID, ErrCannotFindAdapter
	}

	return w.service.GetAdapterGUID()
}

func NewSwiftAdapter(config Config) (*WindowsAdapter, error) {
	adapter := &WindowsAdapter{}
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

		if !config.UnicastIP.IP.IsUnspecified() {
			if err = adapter.SetUnicastIpAddressEntry(&config.UnicastIP); err != nil {
				return nil, err
			}
		}
	case DriverTypeOpenVPN:
		if config.UnicastIP.IP.IsUnspecified() {
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

	if config.DNSConfig != swiftypes.NilDNSConfig {
		if err = adapter.SetDNS(config.DNSConfig); err != nil {
			return nil, err
		}
	}

	return adapter, nil
}
