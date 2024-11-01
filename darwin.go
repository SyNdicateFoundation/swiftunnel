//go:build darwin

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type SwiftInterface struct {
	*tunReadCloser
	name        string
	AdapterType swiftypes.AdapterType
}

const (
	appleUTUNCtl      = "com.apple.net.utun_control"
	appleCTLIOCGINFO  = (0x40000000 | 0x80000000) | ((100 & 0x1fff) << 16) | uint32(byte('N'))<<8 | 3
	sockaddrCtlSize   = 32
	maxAdapterNameLen = 15
)

type sockaddrCtl struct {
	scLen      uint8
	scFamily   uint8
	ssSysaddr  uint16
	scID       uint32
	scUnit     uint32
	scReserved [5]uint32
}

func openDevSystem(config Config) (*SwiftInterface, error) {
	if config.AdapterType != swiftypes.AdapterTypeTUN {
		return nil, errors.New("only TUN is supported for SystemDriver; use TunTapOSXDriver for TAP")
	}

	ifIndex, err := parseUtunIndex(config.AdapterName)
	if err != nil {
		return nil, err
	}

	fd, err := syscall.Socket(syscall.AF_SYSTEM, syscall.SOCK_DGRAM, 2)
	if err != nil {
		return nil, fmt.Errorf("error creating socket: %w", err)
	}

	defer syscall.Close(fd)
	ctlInfo := &struct {
		ctlID   uint32
		ctlName [96]byte
	}{}
	copy(ctlInfo.ctlName[:], appleUTUNCtl)

	if err := ioctl(fd, appleCTLIOCGINFO, uintptr(unsafe.Pointer(ctlInfo))); err != nil {
		return nil, fmt.Errorf("error in ioctl call: %w", err)
	}

	sockAddr := &sockaddrCtl{
		scLen:     sockaddrCtlSize,
		scFamily:  syscall.AF_SYSTEM,
		ssSysaddr: 2,
		scID:      ctlInfo.ctlID,
		scUnit:    uint32(ifIndex) + 1,
	}
	if err := connect(fd, sockAddr); err != nil {
		return nil, fmt.Errorf("error connecting to socket: %w", err)
	}

	ifName, err := getIfName(fd)
	if err != nil {
		return nil, fmt.Errorf("error getting interface name: %w", err)
	}

	return &SwiftInterface{
		name:        ifName,
		AdapterType: config.AdapterType,
		tunReadCloser: &tunReadCloser{
			f: os.NewFile(uintptr(fd), ifName),
		},
	}, nil
}

func parseUtunIndex(name string) (int, error) {
	const utunPrefix = "utun"
	if !strings.HasPrefix(name, utunPrefix) {
		return -1, errors.New("interface name must be in the format utun[0-9]+")
	}

	index, err := strconv.Atoi(name[len(utunPrefix):])
	if err != nil || index < 0 || index > math.MaxUint32-1 {
		return -1, fmt.Errorf("invalid interface index in name: %w", err)
	}
	return index, nil
}

func openDevTunTapOSX(config Config) (*SwiftInterface, error) {
	if len(config.AdapterName) >= maxAdapterNameLen {
		return nil, errors.New("device name is too long")
	}
	prefix := map[swiftypes.AdapterType]string{
		swiftypes.AdapterTypeTUN: "tun",
		swiftypes.AdapterTypeTAP: "tap",
	}
	if expected, ok := prefix[config.AdapterType]; !ok || !strings.HasPrefix(config.AdapterName, expected) {
		return nil, fmt.Errorf("device name must start with %s", expected)
	}

	fd, err := syscall.Open("/dev/"+config.AdapterName, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("error opening device: %w", err)
	}

	if err := setIfUp(fd, config.AdapterName); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return &SwiftInterface{
		name:        config.AdapterName,
		AdapterType: config.AdapterType,
		tunReadCloser: &tunReadCloser{
			f: os.NewFile(uintptr(fd), config.AdapterName),
		},
	}, nil
}

func ioctl(fd int, cmd uint32, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(cmd), arg)
	if errno != 0 {
		return errno
	}
	return nil
}

func connect(fd int, addr *sockaddrCtl) error {
	_, _, errno := syscall.RawSyscall(syscall.SYS_CONNECT, uintptr(fd), uintptr(unsafe.Pointer(addr)), sockaddrCtlSize)
	if errno != 0 {
		return errno
	}
	return nil
}

func getIfName(fd int) (string, error) {
	var ifName [16]byte
	ifNameSize := uintptr(len(ifName))

	_, _, errno := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(fd), 2, 2, uintptr(unsafe.Pointer(&ifName)), uintptr(unsafe.Pointer(&ifNameSize)), 0)
	if errno != 0 {
		return "", errno
	}
	return string(ifName[:ifNameSize-1]), nil
}

func setIfUp(fd int, ifName string) error {
	var ifReq = struct {
		ifName    [16]byte
		ifruFlags int16
		pad       [16]byte
	}{}
	copy(ifReq.ifName[:], ifName)
	ifReq.ifruFlags |= syscall.IFF_RUNNING | syscall.IFF_UP

	return ioctl(fd, syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifReq)))
}

type tunReadCloser struct {
	f    *os.File
	rMu  sync.Mutex
	rBuf []byte
	wMu  sync.Mutex
	wBuf []byte
}

func (t *tunReadCloser) Read(to []byte) (int, error) {
	t.rMu.Lock()
	defer t.rMu.Unlock()

	if cap(t.rBuf) < len(to)+4 {
		t.rBuf = make([]byte, len(to)+4)
	}
	t.rBuf = t.rBuf[:len(to)+4]

	n, err := t.f.Read(t.rBuf)
	copy(to, t.rBuf[4:])
	return n - 4, err
}

func (t *tunReadCloser) Write(from []byte) (int, error) {
	if len(from) == 0 {
		return 0, syscall.EIO
	}

	t.wMu.Lock()
	defer t.wMu.Unlock()

	if cap(t.wBuf) < len(from)+4 {
		t.wBuf = make([]byte, len(from)+4)
	}
	t.wBuf = t.wBuf[:len(from)+4]

	switch ipVer := from[0] >> 4; ipVer {
	case 4:
		t.wBuf[3] = syscall.AF_INET
	case 6:
		t.wBuf[3] = syscall.AF_INET6
	default:
		return 0, errors.New("invalid IP version")
	}

	copy(t.wBuf[4:], from)
	n, err := t.f.Write(t.wBuf)
	return n - 4, err
}

func (t *tunReadCloser) Close() error {
	return t.f.Close()
}

func (a *SwiftInterface) File() *os.File {
	return a.tunReadCloser.f
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
	switch config.DriverType {
	case DriverTypeTunTapOSX:
		osx, err := openDevTunTapOSX(config)
		if config.UnicastIP.IP.IsUnspecified() {
			if err = osx.SetUnicastIpAddressEntry(&config.UnicastIP); err != nil {
				return nil, err
			}
		}

		if config.MTU > 0 {
			if err = osx.SetMTU(config.MTU); err != nil {
				return nil, err
			}
		}

		return osx, err
	case DriverTypeSystem:
		system, err := openDevSystem(config)
		if config.UnicastIP.IP.IsUnspecified() {
			if err = system.SetUnicastIpAddressEntry(&config.UnicastIP); err != nil {
				return nil, err
			}
		}

		if config.MTU > 0 {
			if err = system.SetMTU(config.MTU); err != nil {
				return nil, err
			}
		}
		return system, err
	default:
		return nil, errors.New("unrecognized driver")
	}
}
