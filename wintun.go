//go:build windows

package wintungo

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Wintun DLL functions
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
)

var once sync.Once

func init() {
	once.Do(func() {
		dllFile, err := os.CreateTemp("", "wintun_*.dll")
		if err != nil {
			panic(fmt.Errorf("failed to create temp file for Wintun DLL: %w", err))
		}

		defer os.Remove(dllFile.Name()) // Clean up on exit

		if _, err := dllFile.Write(wintunDLLData); err != nil {
			panic(fmt.Errorf("failed to write Wintun DLL data: %w", err))
		}

		if err := dllFile.Close(); err != nil {
			panic(fmt.Errorf("failed to close temp file: %w", err))
		}

		// Load the Wintun DLL and its functions
		wintunDLL = windows.MustLoadDLL(dllFile.Name())
		loadWintunFunctions()
	})
}

func loadWintunFunctions() {
	var procs = map[string]**windows.Proc{
		"WintunCreateAdapter":           &wintunCreateAdapterFunc,
		"WintunCloseAdapter":            &wintunCloseAdapterFunc,
		"WintunStartSession":            &wintunStartSessionFunc,
		"WintunAllocateSendPacket":      &wintunAllocateSendPacketFunc,
		"WintunEndSession":              &wintunEndSessionFunc,
		"WintunGetRunningDriverVersion": &wintunGetRunningDriverVersionFunc,
		"WintunSendPacket":              &wintunSendPacketFunc,
		"WintunReceivePacket":           &wintunReceivePacketFunc,
		"WintunGetReadWaitEvent":        &wintunGetReadWaitEventFunc,
		"WintunReleaseReceivePacket":    &wintunReleaseReceivePacketFunc,
		"WintunGetAdapterLUID":          &wintunGetAdapterLUIDFunc,
	}
	for name, proc := range procs {
		*proc = wintunDLL.MustFindProc(name)
	}
}

type Adapter struct {
	name       string
	tunnelType string
	handle     uintptr
}

type Session struct {
	handle    uintptr
	waitEvent windows.Handle
}

// NewWintunAdapter creates a new Wintun adapter.
func NewWintunAdapter(name, tunnelType string) (*Adapter, error) {
	return NewWintunAdapterWithGUID(name, tunnelType, windows.GUID{})
}

// NewWintunAdapterWithGUID creates a new Wintun adapter with a specified GUID.
func NewWintunAdapterWithGUID(name, tunnelType string, guid windows.GUID) (*Adapter, error) {
	if name == "" {
		name = "Wintun"
	}

	if tunnelType == "" {
		tunnelType = "VPN Tunnel"
	}

	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("failed to convert adapter name: %w", err)
	}

	tunnelTypePtr, err := windows.UTF16PtrFromString(tunnelType)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tunnel type: %w", err)
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

	return &Adapter{
		name:       name,
		tunnelType: tunnelType,
		handle:     handle,
	}, nil
}

// Close closes the Wintun adapter.
func (a *Adapter) Close() error {
	if _, _, err := wintunCloseAdapterFunc.Call(a.handle); err != nil && !errors.Is(err, windows.NOERROR) {
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
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return nil, fmt.Errorf("failed to create wait event: %v", err)
	}

	return &Session{
		handle:    handle,
		waitEvent: windows.Handle(waitEvent),
	}, nil
}

// Close ends the Wintun session.
func (s *Session) Close() error {
	if _, _, err := wintunEndSessionFunc.Call(s.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to end Wintun session: %v", err)
	}
	return nil
}

// SendPacket sends a packet over the Wintun session.
func (s *Session) SendPacket(packet []byte) error {
	if len(packet) == 0 {
		return errors.New("packet cannot be empty")
	}

	// Allocate memory for the packet in the Wintun session
	dataPtr, _, err := wintunAllocateSendPacketFunc.Call(s.handle, uintptr(len(packet)))
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to allocate Wintun packet: %v", err)
	}

	if dataPtr == 0 {
		return errors.New("allocated data pointer is invalid")
	}

	// Copy packet data to the allocated memory
	dst := (*[1 << 30]byte)(unsafe.Pointer(dataPtr))[:len(packet):len(packet)]
	copy(dst, packet)

	// Send the packet
	if _, _, err := wintunSendPacketFunc.Call(s.handle, dataPtr, uintptr(len(packet))); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to send Wintun packet: %v", err)
	}
	return nil
}

// ReceivePacketNow receives a packet from the Wintun session. Returns nil if no packet is available.
func (s *Session) ReceivePacketNow() ([]byte, error) {
	var packetSize uint32
	packetPtr, _, err := wintunReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetSize)))
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return nil, err
	}

	if packetPtr == 0 {
		return nil, errors.New("failed to receive Wintun packet")
	}

	// Convert packetPtr to a Go slice
	packet := make([]byte, packetSize)
	if packetSize > 0 {
		copy(packet, (*[1 << 30]byte)(unsafe.Pointer(packetPtr))[:packetSize])
	}

	// Release the received packet
	if _, _, err := wintunReleaseReceivePacketFunc.Call(s.handle, packetPtr); err != nil && !errors.Is(err, windows.NOERROR) {
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
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return "", fmt.Errorf("failed to get Wintun driver version: %v", err)
	}

	version := uint32(ret)
	major := (version >> 16) & 0xff
	minor := version & 0xff

	return fmt.Sprintf("%d.%d", major, minor), nil
}

// GetAdapterLUID retrieves the LUID of the adapter.
func (a *Adapter) GetAdapterLUID() (windows.LUID, error) {
	var luid windows.LUID

	if _, _, err := wintunGetAdapterLUIDFunc.Call(
		a.handle,
		uintptr(unsafe.Pointer(&luid)),
	); err != nil && !errors.Is(err, windows.NOERROR) {
		return windows.LUID{}, fmt.Errorf("failed to convert GUID to LUID: %v", err)
	}

	return luid, nil
}

func (a *Adapter) GetAdapterGUID() (windows.GUID, error) {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return windows.GUID{}, err
	}

	var retrievedGUID windows.GUID

	if _, _, err := convertInterfaceLuidToGuid.Call(uintptr(unsafe.Pointer(&luid)), uintptr(unsafe.Pointer(&retrievedGUID))); err != nil && !errors.Is(err, windows.NOERROR) {
		return windows.GUID{}, fmt.Errorf("failed to convert LUID to GUID: %v", err)
	}

	return retrievedGUID, nil
}

func (a *Adapter) GetAdapterIndex() (uint32, error) {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return 0, err
	}

	var index uint32
	if _, _, err := convertLUIDToIndex.Call(uintptr(unsafe.Pointer(&luid)), uintptr(unsafe.Pointer(&index))); err != nil && !errors.Is(err, windows.NOERROR) {
		return 0, fmt.Errorf("failed to convert LUID to index: %v", err)
	}

	return index, nil
}

// GetAdapterName retrieves the name of the adapter.
func (a *Adapter) GetAdapterName() string {
	return a.name
}
