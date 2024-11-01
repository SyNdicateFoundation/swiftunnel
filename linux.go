//go:build linux

package swifttunnel

import (
	"errors"
	"github.com/XenonCommunity/swifttunnel/swiftypes"
	"net"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	cIFFTUN        uint16 = 0x0001
	cIFFTAP        uint16 = 0x0002
	cIFFNOPI       uint16 = 0x1000
	cIFFMULTIQUEUE uint16 = 0x0100
)

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	_     [0x28 - 0x10 - 2]byte // Padding to match struct size
}

type LinuxAdapter struct {
	name        string
	file        *os.File
	AdapterType swiftypes.AdapterType
}

func (a *LinuxAdapter) initializeAdapter(config Config, fd uintptr) (string, error) {
	flags := cIFFNOPI
	if config.AdapterType == swiftypes.AdapterTypeTUN {
		flags |= cIFFTUN
	} else {
		flags |= cIFFTAP
	}

	if config.MultiQueue {
		flags |= cIFFMULTIQUEUE
	}

	ifName, err := a.createInterface(fd, config.AdapterName, flags)
	if err != nil {
		return "", err
	}

	if err := a.setDeviceOptions(fd, config); err != nil {
		return "", err
	}

	return ifName, nil
}

func (a *LinuxAdapter) createInterface(fd uintptr, ifName string, flags uint16) (string, error) {
	var req ifReq
	req.Flags = flags
	copy(req.Name[:], ifName)

	if err := ioctl(fd, syscall.TUNSETIFF, uintptr(unsafe.Pointer(&req))); err != nil {
		return "", err
	}

	return strings.TrimRight(string(req.Name[:]), "\x00"), nil
}

func (a *LinuxAdapter) setDeviceOptions(fd uintptr, config Config) error {
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

func ioctl(fd uintptr, request uintptr, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, argp)
	if errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

func (a *LinuxAdapter) File() *os.File {
	return a.file
}

func (a *LinuxAdapter) GetAdapterName() (string, error) {
	if a.name == "" {
		return "", errors.New("adapter name is not set")
	}
	return a.name, nil
}

func (a *LinuxAdapter) GetAdapterIndex() (uint32, error) {
	if a.name == "" {
		return 0, errors.New("adapter name is not set")
	}

	ifi, err := net.InterfaceByName(a.name)
	if err != nil {
		return 0, err
	}

	return uint32(ifi.Index), nil
}

func (a *LinuxAdapter) SetMTU(mtu int) error {
	return setMTU(a.name, mtu)
}

func (a *LinuxAdapter) SetUnicastIpAddressEntry(entry *net.IPNet) error {
	return setUnicastIpAddressEntry(a.name, entry)
}

func NewSwiftAdapter(config Config) (*LinuxAdapter, error) {
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	adapter := &LinuxAdapter{
		AdapterType: config.AdapterType,
		file:        os.NewFile(uintptr(fd), "tun"),
	}

	adapterName, err := adapter.initializeAdapter(config, uintptr(fd))
	if err != nil {
		adapter.file.Close()
		return nil, err
	}

	adapter.name = adapterName

	if !config.UnicastIP.IP.IsUnspecified() {
		if err = adapter.SetUnicastIpAddressEntry(&config.UnicastIP); err != nil {
			adapter.file.Close()
			return nil, err
		}
	}

	if config.MTU > 0 {
		if err = adapter.SetMTU(config.MTU); err != nil {
			adapter.file.Close()
			return nil, err
		}
	}

	return adapter, nil
}
