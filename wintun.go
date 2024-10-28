//go:build windows

package wintun

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
	
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/windows"
)

/*
#cgo LDFLAGS: -liphlpapi
#include <wintun.h>
#include <iphlpapi.h>

#pragma comment(lib, "iphlpapi.lib")
*/
import "C"

// Wintun DLL functions
var (
	wintunDLL                         *syscall.DLL
	wintunCreateAdapterFunc           *syscall.Proc
	wintunCloseAdapterFunc            *syscall.Proc
	wintunStartSessionFunc            *syscall.Proc
	wintunEndSessionFunc              *syscall.Proc
	wintunGetRunningDriverVersionFunc *syscall.Proc
	wintunSendPacketFunc              *syscall.Proc
	wintunReceivePacketFunc           *syscall.Proc
	wintunReleaseReceivePacketFunc    *syscall.Proc
	wintunGetAdapterLUIDFunc          *syscall.Proc
)

func init() {
	if runtime.GOOS != "windows" {
		panic("Wintun is only supported on Windows")
	}
	
	// Create a temporary DLL file
	dllFile, err := os.CreateTemp("", "wintun_*.dll")
	if err != nil {
		panic(fmt.Errorf("failed to create temp file for Wintun DLL: %w", err))
	}
	
	defer os.Remove(dllFile.Name()) // Clean up on exit
	
	// Write the Wintun DLL data to the temporary file
	if _, err := dllFile.Write(wintunDLLData); err != nil {
		panic(fmt.Errorf("failed to write Wintun DLL data: %w", err))
	}
	
	if err := dllFile.Close(); err != nil {
		panic(fmt.Errorf("failed to close temp file: %w", err))
	}
	
	// Load the Wintun DLL
	wintunDLL = syscall.MustLoadDLL(dllFile.Name())
	
	// Load the required functions from the DLL
	loadWintunFunctions()
}

func loadWintunFunctions() {
	wintunCreateAdapterFunc = wintunDLL.MustFindProc("WintunCreateAdapter")
	wintunCloseAdapterFunc = wintunDLL.MustFindProc("WintunCloseAdapter")
	wintunStartSessionFunc = wintunDLL.MustFindProc("WintunStartSession")
	wintunEndSessionFunc = wintunDLL.MustFindProc("WintunEndSession")
	wintunGetRunningDriverVersionFunc = wintunDLL.MustFindProc("WintunGetRunningDriverVersion")
	wintunSendPacketFunc = wintunDLL.MustFindProc("WintunSendPacket")
	wintunReceivePacketFunc = wintunDLL.MustFindProc("WintunReceivePacket")
	wintunReleaseReceivePacketFunc = wintunDLL.MustFindProc("WintunReleaseReceivePacket")
	wintunGetAdapterLUIDFunc = wintunDLL.MustFindProc("WintunGetAdapterLUID")
}

type LUID C.NET_LUID

type Adapter struct {
	Name       string
	TunnelType string
	handle     uintptr
}

type Session struct {
	handle uintptr
}

// NewWintunAdapter creates a new Wintun adapter with the specified name and tunnel type.
func NewWintunAdapter(name, tunnelType string) (*Adapter, error) {
	return NewWintunAdapterWithGUID(name, tunnelType, syscall.GUID{})
}

// NewWintunAdapterWithGUID creates a new Wintun adapter with a specified GUID.
func NewWintunAdapterWithGUID(name, tunnelType string, guid syscall.GUID) (*Adapter, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	
	tunnelTypePtr, err := syscall.UTF16PtrFromString(tunnelType)
	if err != nil {
		return nil, err
	}
	
	var handle uintptr
	if guid != (syscall.GUID{}) {
		handle, _, err = wintunCreateAdapterFunc.Call(
			uintptr(unsafe.Pointer(namePtr)),
			uintptr(unsafe.Pointer(tunnelTypePtr)),
			uintptr(unsafe.Pointer(&guid)),
		)
	} else {
		handle, _, err = wintunCreateAdapterFunc.Call(
			uintptr(unsafe.Pointer(namePtr)),
			uintptr(unsafe.Pointer(tunnelTypePtr)),
			0,
		)
	}
	
	if handle == 0 {
		return nil, fmt.Errorf("failed to create Wintun adapter: %v", err)
	}
	
	return &Adapter{handle: handle, Name: name, TunnelType: tunnelType}, nil
}

// Close closes the Wintun adapter.
func (a *Adapter) Close() error {
	_, _, err := wintunCloseAdapterFunc.Call(a.handle)
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("failed to close Wintun adapter: %v", err)
	}
	return nil
}

// StartSession starts a new session on the Wintun adapter.
func (a *Adapter) StartSession(capacity uint32) (*Session, error) {
	handle, _, err := wintunStartSessionFunc.Call(a.handle, uintptr(capacity))
	if handle == 0 {
		return nil, fmt.Errorf("failed to start Wintun session: %v", err)
	}
	return &Session{handle: handle}, nil
}

// Close ends the Wintun session.
func (s *Session) Close() error {
	_, _, err := wintunEndSessionFunc.Call(s.handle)
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("failed to end Wintun session: %v", err)
	}
	return nil
}

// SendPacket sends a packet over the Wintun session.
func (s *Session) SendPacket(packet []byte) error {
	packetPtr := unsafe.Pointer(&packet[0])
	_, _, err := wintunSendPacketFunc.Call(s.handle, uintptr(packetPtr), uintptr(len(packet)))
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("failed to send Wintun packet: %v", err)
	}
	return nil
}

// ReceivePacket receives a packet from the Wintun session.
func (s *Session) ReceivePacket() ([]byte, error) {
	var packetSize uint32
	var packetPtr unsafe.Pointer
	
	_, _, err := wintunReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetPtr)), uintptr(unsafe.Pointer(&packetSize)))
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return nil, fmt.Errorf("failed to receive Wintun packet: %v", err)
	}
	
	packet := C.GoBytes(packetPtr, C.int(packetSize))
	
	// Release the received packet
	_, _, releaseErr := wintunReleaseReceivePacketFunc.Call(s.handle, uintptr(packetPtr))
	if releaseErr != nil && !errors.Is(releaseErr, syscall.Errno(0)) {
		return nil, fmt.Errorf("failed to release Wintun packet: %v", releaseErr)
	}
	
	return packet, nil
}

// GetRunningDriverVersion retrieves the running version of the Wintun driver.
func (a *Adapter) GetRunningDriverVersion() (string, error) {
	ret, _, err := wintunGetRunningDriverVersionFunc.Call()
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return "", fmt.Errorf("failed to get running Wintun driver version: %v", err)
	}
	
	version := uint32(ret)
	major := (version >> 16) & 0xff
	minor := version & 0xff
	return fmt.Sprintf("%d.%d", major, minor), nil
}

// GetAdapterLUID retrieves the LUID of the Wintun adapter.
func (a *Adapter) GetAdapterLUID() (LUID, error) {
	var luid LUID
	
	ret, _, err := wintunGetAdapterLUIDFunc.Call(uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(a.Name))), uintptr(unsafe.Pointer(&luid)))
	if ret != 0 {
		return luid, fmt.Errorf("failed to get adapter LUID: %w", err)
	}
	
	return luid, nil
}

// AddrAdd adds an address to the Wintun adapter.
func (a *Adapter) AddrAdd(addr *netlink.Addr) error {
	link, err := a.GetLink()
	if err != nil {
		return fmt.Errorf("failed to get link by name: %w", err)
	}
	return netlink.AddrAdd(link, addr)
}

// AddrDel deletes an address from the Wintun adapter.
func (a *Adapter) AddrDel(addr *netlink.Addr) error {
	link, err := a.GetLink()
	if err != nil {
		return fmt.Errorf("failed to get link by name: %w", err)
	}
	return netlink.AddrDel(link, addr)
}

// AddrList retrieves the list of addresses for the Wintun adapter.
func (a *Adapter) AddrList() ([]netlink.Addr, error) {
	link, err := a.GetLink()
	if err != nil {
		return nil, fmt.Errorf("failed to get link by name: %w", err)
	}
	return netlink.AddrList(link, syscall.AF_INET)
}

// GetLink retrieves the link for the Wintun adapter.
func (a *Adapter) GetLink() (netlink.Link, error) {
	link, err := netlink.LinkByName(a.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get link: %w", err)
	}
	return link, nil
}
