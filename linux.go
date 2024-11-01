//go:build linux

package swiftunnel

import (
	"errors"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"io"
	"log"
	"net"
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

type SwiftInterface struct {
	io.ReadWriteCloser
	name        string
	adapterType swiftypes.AdapterType
}

func (a *SwiftInterface) initializeAdapter(config Config, fd uintptr) (string, error) {
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

func (a *SwiftInterface) createInterface(fd uintptr, ifName string, flags uint16) (string, error) {
	var req ifReq

	req.Flags = flags
	copy(req.Name[:], ifName)

	if err := ioctl(fd, syscall.TUNSETIFF, uintptr(unsafe.Pointer(&req))); err != nil {
		return "", err
	}

	return strings.TrimRight(string(req.Name[:]), "\x00"), nil
}

func (a *SwiftInterface) setDeviceOptions(fd uintptr, config Config) error {
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

func (a *SwiftInterface) GetFD() *os.File {
	return a.ReadWriteCloser.(*os.File)
}

func (a *SwiftInterface) GetAdapterName() (string, error) {
	if a.name == "" {
		return "", errors.New("adapter name is not set")
	}
	return a.name, nil
}

func (a *SwiftInterface) GetAdapterIndex() (uint32, error) {
	if a.name == "" {
		return 0, errors.New("adapter name is not set")
	}

	ifi, err := net.InterfaceByName(a.name)
	if err != nil {
		return 0, err
	}

	return uint32(ifi.Index), nil
}

func (a *SwiftInterface) SetMTU(mtu int) error {
	return setMTU(a.name, mtu)
}

func (a *SwiftInterface) SetUnicastIpAddressEntry(entry *net.IPNet) error {
	return setUnicastIpAddressEntry(a.name, entry)
}

func NewSwiftInterface(config Config) (*SwiftInterface, error) {
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	adapter := &SwiftInterface{
		adapterType:     config.AdapterType,
		ReadWriteCloser: os.NewFile(uintptr(fd), "tun"),
	}

	adapterName, err := adapter.initializeAdapter(config, uintptr(fd))
	if err != nil {
		adapter.Close()
		return nil, err
	}

	adapter.name = adapterName

	if config.UnicastIP != nil {
		log.Printf("Setting unicast IP address: %v", config.UnicastIP)
		if err = adapter.SetUnicastIpAddressEntry(config.UnicastIP); err != nil {
			adapter.Close()
			return nil, err
		}
	}

	if config.MTU > 0 {
		if err = adapter.SetMTU(config.MTU); err != nil {
			adapter.Close()
			return nil, err
		}
	}

	return adapter, nil
}
