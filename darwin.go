//go:build darwin

package swiftunnel

import (
	"errors"
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel/swiftconfig"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

// SwiftInterface represents a macOS network tunnel interface.
type SwiftInterface struct {
	*tunReadCloser
	name        string
	AdapterType swiftypes.AdapterType
	index       int
}

const (
	appleUTUNCtl       = "com.apple.net.utun_control"
	appleCTLIOCGINFO   = (0x40000000 | 0x80000000) | ((100 & 0x1fff) << 16) | uint32(byte('N'))<<8 | 3
	sockaddrCtlSize    = 32
	maxAdapterNameLen  = 15
	internalBufferSize = 2048
)

type sockaddrCtl struct {
	scLen      uint8
	scFamily   uint8
	ssSysaddr  uint16
	scID       uint32
	scUnit     uint32
	scReserved [5]uint32
}

// openDevSystem initializes a native macOS utun interface.
func openDevSystem(config *swiftconfig.Config) (*SwiftInterface, error) {
	if config.AdapterType != swiftypes.AdapterTypeTUN {
		return nil, errors.New("only TUN is supported for SystemDriver; use TunTapOSXDriver for TAP")
	}

	ifIndex, err := parseUtunIndex(config.AdapterName)
	if err != nil {
		return nil, err
	}

	fd, err := unix.Socket(unix.AF_SYSTEM, unix.SOCK_DGRAM, 2)
	if err != nil {
		return nil, fmt.Errorf("error creating socket: %w", err)
	}

	defer func(fd int) {
		_ = unix.Close(fd)
	}(fd)

	ctlInfo := &struct {
		ctlID   uint32
		ctlName [96]byte
	}{}

	copy(ctlInfo.ctlName[:], appleUTUNCtl)

	if err := ioctl(uintptr(fd), uintptr(appleCTLIOCGINFO), uintptr(unsafe.Pointer(ctlInfo))); err != nil {
		return nil, fmt.Errorf("error in ioctl call: %w", err)
	}

	sockAddr := &sockaddrCtl{
		scLen:     sockaddrCtlSize,
		scFamily:  unix.AF_SYSTEM,
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
		index:       ifIndex,
		AdapterType: config.AdapterType,
		tunReadCloser: &tunReadCloser{
			f: os.NewFile(uintptr(fd), ifName),
		},
	}, nil
}

// parseUtunIndex extracts the integer index from an utun device name.
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

// openDevTunTapOSX initializes a third-party TunTapOSX driver interface.
func openDevTunTapOSX(config *swiftconfig.Config) (*SwiftInterface, error) {
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

	fd, err := unix.Open("/dev/"+config.AdapterName, os.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("error opening device: %w", err)
	}

	if err := setIfUp(uintptr(fd), config.AdapterName); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	name, err := netlink.LinkByName(config.AdapterName)
	if err != nil {
		return nil, err
	}

	return &SwiftInterface{
		name:        config.AdapterName,
		index:       name.Attrs().Index,
		AdapterType: config.AdapterType,
		tunReadCloser: &tunReadCloser{
			f: os.NewFile(uintptr(fd), config.AdapterName),
		},
	}, nil
}

// connect performs a system call to connect the control socket.
func connect(fd int, addr *sockaddrCtl) error {
	_, _, errno := unix.RawSyscall(unix.SYS_CONNECT, uintptr(fd), uintptr(unsafe.Pointer(addr)), sockaddrCtlSize)
	if errno != 0 {
		return errno
	}
	return nil
}

// getIfName retrieves the actual interface name assigned by the system.
func getIfName(fd int) (string, error) {
	var ifName [16]byte
	ifNameSize := uintptr(len(ifName))

	_, _, errno := unix.Syscall6(unix.SYS_GETSOCKOPT, uintptr(fd), 2, 2, uintptr(unsafe.Pointer(&ifName)), uintptr(unsafe.Pointer(&ifNameSize)), 0)
	if errno != 0 {
		return "", errno
	}

	return string(ifName[:ifNameSize-1]), nil
}

// setIfUp configures the interface status to UP and RUNNING.
func setIfUp(fd uintptr, ifName string) error {
	var ifReq = struct {
		ifName    [16]byte
		ifruFlags int16
		pad       [16]byte
	}{}

	copy(ifReq.ifName[:], ifName)
	ifReq.ifruFlags |= unix.IFF_RUNNING | unix.IFF_UP

	return ioctl(fd, unix.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifReq)))
}

type tunReadCloser struct {
	f *os.File
}

// Read implements the io.Reader interface for the macOS tunnel, handling the 4-byte PI header.
func (t *tunReadCloser) Read(to []byte) (int, error) {
	buf := make([]byte, internalBufferSize)

	n, err := t.f.Read(buf)
	if err != nil {
		return 0, err
	}

	if n <= 4 {
		return 0, nil
	}

	payloadLen := n - 4
	if len(to) < payloadLen {
		return 0, io.ErrShortBuffer
	}

	copy(to, buf[4:n])

	return payloadLen, nil
}

// Write implements the io.Writer interface for the macOS tunnel, prepending the 4-byte PI header.
func (t *tunReadCloser) Write(from []byte) (int, error) {
	if len(from) == 0 {
		return 0, nil
	}

	var proto uint32
	switch from[0] >> 4 {
	case 4:
		proto = unix.AF_INET
	case 6:
		proto = unix.AF_INET6
	default:
		return 0, errors.New("unknown ip version")
	}

	totalLen := len(from) + 4
	buf := make([]byte, totalLen)

	buf[0] = 0
	buf[1] = 0
	buf[2] = 0
	buf[3] = byte(proto)

	copy(buf[4:], from)

	n, err := t.f.Write(buf)
	if n >= 4 {
		return n - 4, err
	}
	return 0, err
}

// Close releases the underlying file descriptor.
func (t *tunReadCloser) Close() error {
	return t.f.Close()
}

// GetFD returns the underlying OS file object.
func (a *SwiftInterface) GetFD() *os.File {
	return a.tunReadCloser.f
}

// NewSwiftInterface creates and configures a new macOS SwiftInterface based on the driver type.
func NewSwiftInterface(config *swiftconfig.Config) (*SwiftInterface, error) {
	switch config.DriverType {
	case swiftconfig.DriverTypeTunTapOSX:
		tapOSX, err := openDevTunTapOSX(config)
		if config.UnicastConfig == nil {
			if err = tapOSX.SetUnicastIpAddressEntry(config.UnicastConfig); err != nil {
				return nil, err
			}
		}

		if config.MTU > 0 {
			if err = tapOSX.SetMTU(config.MTU); err != nil {
				return nil, err
			}
		}

		return tapOSX, err
	case swiftconfig.DriverTypeSystem:
		system, err := openDevSystem(config)
		if config.UnicastConfig == nil {
			if err = system.SetUnicastIpAddressEntry(config.UnicastConfig); err != nil {
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
