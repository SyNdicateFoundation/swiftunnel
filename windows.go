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

type WindowsAdapter struct {
	wintunAdapter *wintun.WintunAdapter
	wintunSession *wintun.WintunSession
	ovpn          *openvpn.OpenVPNAdapter
}

func (w *WindowsAdapter) Write(buf []byte) (int, error) {
	switch {
	case w.wintunSession != nil:
		return w.wintunSession.Write(buf)
	case w.ovpn != nil:
		return w.ovpn.Write(buf)
	default:
		return 0, ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) Read(buf []byte) (int, error) {
	switch {
	case w.wintunSession != nil:
		return w.wintunSession.Read(buf)
	case w.ovpn != nil:
		return w.ovpn.Read(buf)
	default:
		return 0, ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) Close() error {
	switch {
	case w.wintunSession != nil:
		return w.wintunSession.Close()
	case w.ovpn != nil:
		return w.ovpn.Close()
	default:
		return ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) File() *os.File {
	switch {
	case w.ovpn != nil:
		return w.ovpn.File()
	default:
		return nil
	}
}

func (w *WindowsAdapter) GetAdapterName() (string, error) {
	switch {
	case w.wintunAdapter != nil:
		return w.wintunAdapter.GetAdapterName()
	case w.ovpn != nil:
		return w.ovpn.GetAdapterName()
	default:
		return "", ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) GetAdapterIndex() (uint32, error) {
	switch {
	case w.wintunAdapter != nil:
		return w.wintunAdapter.GetAdapterIndex()
	case w.ovpn != nil:
		return w.ovpn.GetAdapterIndex()
	default:
		return 0, ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) SetMTU(mtu uint32) error {
	adapterIndex, err := w.getAdapterIndex()
	if err != nil {
		return err
	}

	return setMTU(adapterIndex, mtu)
}

func (w *WindowsAdapter) SetUnicastIpAddressEntry(entry *net.IPNet) error {
	luid, err := w.getAdapterLUID()
	if err != nil {
		return err
	}

	return setUnicastIpAddressEntry(luid, entry, IpDadStatePreferred)
}

func (w *WindowsAdapter) SetDNS(config *swiftypes.DNSConfig) error {
	guid, err := w.getAdapterGUID()
	if err != nil {
		return err
	}

	return setDNS(guid, config)
}

func (w *WindowsAdapter) getAdapterIndex() (uint32, error) {
	switch {
	case w.wintunSession != nil:
		return w.wintunAdapter.GetAdapterIndex()
	case w.ovpn != nil:
		return w.ovpn.GetAdapterIndex()
	default:
		return 0, ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) getAdapterLUID() (swiftypes.LUID, error) {
	switch {
	case w.wintunSession != nil:
		return w.wintunAdapter.GetAdapterLUID()
	case w.ovpn != nil:
		return w.ovpn.GetAdapterLUID()
	default:
		return swiftypes.NilLUID, ErrCannotFindAdapter
	}
}

func (w *WindowsAdapter) getAdapterGUID() (swiftypes.GUID, error) {
	switch {
	case w.wintunSession != nil:
		return w.wintunAdapter.GetAdapterGUID()
	case w.ovpn != nil:
		return w.ovpn.GetAdapterGUID()
	default:
		return swiftypes.NilGUID, ErrCannotFindAdapter
	}
}

func NewWindowsAdapter(config Config) *WindowsAdapter {
	adapter := &WindowsAdapter{}
	var err error

	switch config.AdapterType {
	case swiftypes.AdapterTypeTUN:
		adapter.wintunAdapter, err = wintun.NewWintunAdapterWithGUID(config.AdapterName, "VPN Tunnel", config.AdapterGUID)
		if err != nil {
			return nil
		}
		adapter.wintunSession, err = adapter.wintunAdapter.StartSession(0x800000)
	case swiftypes.AdapterTypeTAP:
		adapter.ovpn, err = openvpn.NewOpenVPNAdapter(config.AdapterGUID, config.AdapterName, config.UnicastIP.IP, config.UnicastIP, false)
	default:
		return nil
	}

	if err != nil || adapter == nil {
		return nil
	}

	if config.MTU != 0 {
		if err = adapter.SetMTU(config.MTU); err != nil {
			return nil
		}
	}

	if !config.UnicastIP.IP.IsUnspecified() {
		if err = adapter.SetUnicastIpAddressEntry(&config.UnicastIP); err != nil {
			return nil
		}
	}

	if config.DNSConfig != swiftypes.NilDNSConfig {
		if err = adapter.SetDNS(config.DNSConfig); err != nil {
			return nil
		}
	}

	return adapter
}
