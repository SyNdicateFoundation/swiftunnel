//go:build linux

package swiftunnel

import (
	"github.com/SyNdicateFoundation/swiftunnel/swiftconfig"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

type ifReq struct {
	Name  [syscall.IFNAMSIZ]byte
	Flags uint16
	_     [0x28 - 0x10 - 2]byte
}

// SwiftInterface represents a Linux TUN/TAP device.
type SwiftInterface struct {
	io.ReadWriteCloser
	name        string
	adapterType swiftypes.AdapterType
}

// initializeAdapter configures the flags and creates the Linux interface via ioctl.
func (a *SwiftInterface) initializeAdapter(config *swiftconfig.Config, fd uintptr) (string, error) {
	flags := syscall.IFF_NO_PI

	if config.AdapterType == swiftypes.AdapterTypeTUN {
		flags |= syscall.IFF_TUN
	} else {
		flags |= syscall.IFF_TAP
	}

	if config.MultiQueue {
		flags |= syscall.IFF_PROMISC
	}

	ifName, err := a.createInterface(fd, config.AdapterName, uint16(flags))
	if err != nil {
		return "", err
	}

	if err := a.setDeviceOptions(fd, config); err != nil {
		return "", err
	}

	return ifName, nil
}

// createInterface sends the TUNSETIFF ioctl to create the virtual device.
func (a *SwiftInterface) createInterface(fd uintptr, ifName string, flags uint16) (string, error) {
	var req ifReq

	req.Flags = flags
	copy(req.Name[:], ifName)

	if err := ioctl(fd, syscall.TUNSETIFF, uintptr(unsafe.Pointer(&req))); err != nil {
		return "", err
	}

	return strings.TrimRight(string(req.Name[:]), "\x00"), nil
}

// setDeviceOptions configures persistence and ownership permissions.
func (a *SwiftInterface) setDeviceOptions(fd uintptr, config *swiftconfig.Config) error {
	if config.Permissions != nil {
		if err := ioctl(fd, syscall.TUNSETOWNER, uintptr(config.Permissions.Owner)); err != nil {
			return err
		}
		if err := ioctl(fd, syscall.TUNSETGROUP, uintptr(config.Permissions.Group)); err != nil {
			return err
		}
	}

	persistFlag := 0
	if config.Persist {
		persistFlag = 1
	}

	return ioctl(fd, syscall.TUNSETPERSIST, uintptr(persistFlag))
}

// GetFD returns the underlying OS file pointer.
func (a *SwiftInterface) GetFD() *os.File {
	return a.ReadWriteCloser.(*os.File)
}

// NewSwiftInterface opens /dev/net/tun and initializes the SwiftInterface.
func NewSwiftInterface(config *swiftconfig.Config) (*SwiftInterface, error) {
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	adapter := &SwiftInterface{
		adapterType:     config.AdapterType,
		ReadWriteCloser: os.NewFile(uintptr(fd), "tun"),
	}

	adapterName, err := adapter.initializeAdapter(config, uintptr(fd))
	if err != nil {
		_ = adapter.Close()
		return nil, err
	}

	adapter.name = adapterName

	if config.UnicastConfig != nil {
		if err = adapter.SetUnicastIpAddressEntry(config.UnicastConfig); err != nil {
			_ = adapter.Close()
			return nil, err
		}
	}

	if config.MTU > 0 {
		if err = adapter.SetMTU(config.MTU); err != nil {
			_ = adapter.Close()
			return nil, err
		}
	}

	return adapter, nil
}
