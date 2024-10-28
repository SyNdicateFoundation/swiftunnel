//go:build windows

package wintun

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"unsafe"

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
//
//goland:noinspection GoErrorStringFormat
var (
	wintunDLL                         *windows.DLL
	wintunCreateAdapterFunc           *windows.Proc
	wintunCloseAdapterFunc            *windows.Proc
	wintunStartSessionFunc            *windows.Proc
	wintunAllocateSendPacketFunc      *windows.Proc
	wintunEndSessionFunc              *windows.Proc
	wintunGetRunningDriverVersionFunc *windows.Proc
	wintunSendPacketFunc              *windows.Proc
	wintunReceivePacketFunc           *windows.Proc
	wintunGetReadWaitEventFunc        *windows.Proc
	wintunReleaseReceivePacketFunc    *windows.Proc
	wintunGetAdapterLUIDFunc          *windows.Proc
	errZero                           = windows.Errno(0)
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
	wintunDLL = windows.MustLoadDLL(dllFile.Name())

	// Load the required functions from the DLL
	loadWintunFunctions()
}

func loadWintunFunctions() {
	wintunCreateAdapterFunc = wintunDLL.MustFindProc("WintunCreateAdapter")
	wintunCloseAdapterFunc = wintunDLL.MustFindProc("WintunCloseAdapter")
	wintunStartSessionFunc = wintunDLL.MustFindProc("WintunStartSession")
	wintunAllocateSendPacketFunc = wintunDLL.MustFindProc("WintunAllocateSendPacket")
	wintunEndSessionFunc = wintunDLL.MustFindProc("WintunEndSession")
	wintunGetRunningDriverVersionFunc = wintunDLL.MustFindProc("WintunGetRunningDriverVersion")
	wintunSendPacketFunc = wintunDLL.MustFindProc("WintunSendPacket")
	wintunReceivePacketFunc = wintunDLL.MustFindProc("WintunReceivePacket")
	wintunGetReadWaitEventFunc = wintunDLL.MustFindProc("WintunGetReadWaitEvent")
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
	handle    uintptr
	waitEvent windows.Handle
}

// NewWintunAdapter creates a new Wintun adapter with the specified name and tunnel type.
func NewWintunAdapter(name, tunnelType string) (*Adapter, error) {
	return NewWintunAdapterWithGUID(name, tunnelType, windows.GUID{})
}

// NewWintunAdapterWithGUID creates a new Wintun adapter with a specified GUID.
func NewWintunAdapterWithGUID(name, tunnelType string, guid windows.GUID) (*Adapter, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	tunnelTypePtr, err := windows.UTF16PtrFromString(tunnelType)
	if err != nil {
		return nil, err
	}

	var handle uintptr
	if guid != (windows.GUID{}) {
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
	if err != nil && !errors.Is(err, errZero) {
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

	waitEvent, _, err := wintunGetReadWaitEventFunc.Call(handle)
	if err != nil && !errors.Is(err, errZero) {
		return nil, fmt.Errorf("failed to create wait event: %v", err)
	}

	return &Session{
		handle:    handle,
		waitEvent: windows.Handle(waitEvent),
	}, nil
}

// Close ends the Wintun session.
func (s *Session) Close() error {
	_, _, err := wintunEndSessionFunc.Call(s.handle)
	if err != nil && !errors.Is(err, errZero) {
		return fmt.Errorf("failed to end Wintun session: %v", err)
	}
	return nil
}

// SendPacket sends a packet over the Wintun session.
func (s *Session) SendPacket(packet []byte) error {
	if len(packet) == 0 {
		return fmt.Errorf("packet cannot be empty")
	}

	// Allocate memory for the packet in the Wintun session
	data, _, err := wintunAllocateSendPacketFunc.Call(s.handle, uintptr(len(packet)))
	if err != nil && !errors.Is(err, errZero) {
		return fmt.Errorf("failed to allocate Wintun packet: %v", err)
	}

	if data != 0 {
		C.memcpy(unsafe.Pointer(data), unsafe.Pointer(&packet[0]), C.size_t(len(packet)))
	} else {
		return fmt.Errorf("allocated data pointer is invalid")
	}

	// Send the packet
	_, _, err = wintunSendPacketFunc.Call(s.handle, data, uintptr(len(packet)))
	if err != nil && !errors.Is(err, errZero) {
		return fmt.Errorf("failed to send Wintun packet: %v", err)
	}

	return nil
}

// ReceivePacketNow receives a packet from the Wintun session. If no packet is available, it returns nil.
func (s *Session) ReceivePacketNow() ([]byte, error) {
	var packetSize C.DWORD
	var packetPtr *C.BYTE // Pointer to C.BYTE

	// Call the Wintun receive packet function
	ret, _, err := wintunReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetPtr)), uintptr(unsafe.Pointer(&packetSize)))
	if err != nil && !errors.Is(err, errZero) {
		return nil, err
	}

	// Check if the return value indicates success (assuming non-zero indicates success)
	if ret == 0 {
		return nil, fmt.Errorf("failed to receive Wintun packet: %v", err)
	}

	log.Println("Packet size:", packetSize)

	// Convert packetPtr to a Go slice using C.GoBytes
	packet := C.GoBytes(unsafe.Pointer(packetPtr), C.int(packetSize))

	// Release the received packet
	_, _, err = wintunReleaseReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetPtr)))
	if err != nil && !errors.Is(err, errZero) {
		return nil, err
	}

	return packet, nil
}

// ReceivePacket waits for the next packet to be available and then returns it.
func (s *Session) ReceivePacket() ([]byte, error) {
	packet, err := s.ReceivePacketNow()
	if err != nil && err.Error() == "No more data is available." {
		result, err := windows.WaitForSingleObject(s.waitEvent, windows.INFINITE)
		if err != nil {
			return nil, fmt.Errorf("failed to wait for packet: %v", err)
		}

		if result != windows.WAIT_OBJECT_0 {
			return nil, fmt.Errorf("wait event failed: unexpected result %v", result)
		}

		// After waiting, try receiving the packet again
		return s.ReceivePacket()
	}

	return packet, err
}

// GetRunningDriverVersion retrieves the running version of the Wintun driver.
func (a *Adapter) GetRunningDriverVersion() (string, error) {
	ret, _, err := wintunGetRunningDriverVersionFunc.Call()
	if err != nil && !errors.Is(err, errZero) {
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

	_, _, err := wintunGetAdapterLUIDFunc.Call(a.handle, uintptr(unsafe.Pointer(&luid)))
	if err != nil && !errors.Is(err, errZero) {
		return luid, fmt.Errorf("failed to get adapter LUID: %w", err)
	}

	return luid, nil
}
