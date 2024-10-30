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

var (
	once       sync.Once
	errInitDLL error
)

//goland:noinspection GoErrorStringFormat
var (
	ErrInvalidAdapterHandle = errors.New("adapter handle is invalid")
	ErrInvalidSessionHandle = errors.New("session handle is invalid")
	ErrEmptyPacket          = errors.New("packet cannot be empty")
	ErrNoDataAvailable      = errors.New("No more data is available.")
	ErrFailedToReceive      = errors.New("failed to receive Wintun packet")
)

func init() {
	once.Do(func() {
		errInitDLL = loadWintunDLL()
	})
}

func loadWintunDLL() error {
	dllFile, err := os.CreateTemp("", "wintun_*.dll")
	if err != nil {
		return fmt.Errorf("failed to create temp file for Wintun DLL: %w", err)
	}

	defer os.Remove(dllFile.Name())

	if _, err := dllFile.Write(wintunDLLData); err != nil {
		return fmt.Errorf("failed to write Wintun DLL data: %w", err)
	}
	if err := dllFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Load the Wintun DLL and its functions
	wintunDLL = windows.MustLoadDLL(dllFile.Name())
	return loadWintunFunctions()
}

func loadWintunFunctions() error {
	procs := map[string]**windows.Proc{
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
		if *proc == nil {
			return fmt.Errorf("failed to load procedure %s", name)
		}
	}
	return nil
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

// NewWintunAdapter creates a new Wintun adapter with the given name and
// tunnelType. The adapter's GUID is chosen by Windows.
//
// The returned Adapter object can be used to start a session with
// StartSession.
func NewWintunAdapter(name, tunnelType string) (*Adapter, error) {
	return NewWintunAdapterWithGUID(name, tunnelType, windows.GUID{})
}

// NewWintunAdapterWithGUID creates a new Wintun adapter with the given name,
// tunnelType and GUID.
//
// The GUID is optional and can be set to the zero value to let Windows choose
// a GUID for the adapter.
//
// The returned Adapter object can be used to start a session with
// WintunStartSession.
//
// The returned error is nil on success, otherwise it contains an error message.
func NewWintunAdapterWithGUID(name, tunnelType string, guid windows.GUID) (*Adapter, error) {
	if errInitDLL != nil {
		return nil, fmt.Errorf("wintun DLL initialization failed: %w", errInitDLL)
	}

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

// Close releases Wintun adapter resources and, if adapter was created with
// NewWintunAdapter, removes adapter.
//
// If the function fails, the return value is an error. To get extended error
// information, call the Windows API function GetLastError.
func (a *Adapter) Close() error {
	if a.handle == 0 {
		return ErrInvalidAdapterHandle
	}

	if _, _, err := wintunCloseAdapterFunc.Call(a.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to close Wintun adapter: %v", err)
	}
	return nil
}

// StartSession initiates a new Wintun session with the specified capacity.
func (a *Adapter) StartSession(capacity uint32) (*Session, error) {
	if a.handle == 0 {
		return nil, ErrInvalidAdapterHandle
	}

	handle, _, err := wintunStartSessionFunc.Call(a.handle, uintptr(capacity))
	if handle == 0 {
		return nil, fmt.Errorf("failed to start Wintun session: %v", err)
	}

	waitEvent, _, err := wintunGetReadWaitEventFunc.Call(handle)
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return nil, fmt.Errorf("failed to create wait event: %v", err)
	}

	return &Session{handle: handle, waitEvent: windows.Handle(waitEvent)}, nil
}

// Close ends the Wintun session.
//
// The function will return an error if the session handle is invalid or if there is an error ending the session.
func (s *Session) Close() error {
	if s.handle == 0 {
		return ErrInvalidSessionHandle
	}

	if _, _, err := wintunEndSessionFunc.Call(s.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to end Wintun session: %v", err)
	}
	return nil
}

// SendPacket sends a packet over the Wintun session.
//
// The function will return an error if the session handle is invalid, the packet is empty, or if there is an error allocating or sending the packet.
//
// The packet is copied to internal memory and may be modified or reused immediately.
func (s *Session) SendPacket(packet []byte) error {
	if s.handle == 0 {
		return ErrInvalidSessionHandle
	}

	if len(packet) == 0 {
		return ErrEmptyPacket
	}

	dataPtr, _, err := wintunAllocateSendPacketFunc.Call(s.handle, uintptr(len(packet)))
	if (err != nil && !errors.Is(err, windows.NOERROR)) || dataPtr == 0 {
		return fmt.Errorf("failed to allocate Wintun packet: %v", err)
	}

	copy((*[1 << 30]byte)(unsafe.Pointer(dataPtr))[:len(packet):len(packet)], packet)

	// Send the packet
	if _, _, err := wintunSendPacketFunc.Call(s.handle, dataPtr, uintptr(len(packet))); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to send Wintun packet: %v", err)
	}

	return nil
}

// ReceivePacketNow receives a packet from the Wintun session. If no packet is available, returns an error.
//
// The function will return immediately if the packet queue is not empty. If the packet queue is empty, the function will
// return an error. If the session handle is invalid or a critical error occurs, the function will return an error.
//
// The returned packet is a copy of the internal packet buffer and may be modified or reused immediately.
func (s *Session) ReceivePacketNow() ([]byte, error) {
	if s.handle == 0 {
		return nil, ErrInvalidSessionHandle
	}

	var packetSize uint32
	packetPtr, _, err := wintunReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetSize)))
	if packetPtr == 0 || !errors.Is(err, windows.NOERROR) {
		if err != nil {
			if err.Error() == "No more data is available." {
				return nil, ErrNoDataAvailable
			}
		} else {
			return nil, ErrFailedToReceive
		}
	}

	packet := make([]byte, packetSize)
	copy(packet, (*[1 << 30]byte)(unsafe.Pointer(packetPtr))[:packetSize:packetSize])

	if _, _, err := wintunReleaseReceivePacketFunc.Call(s.handle, packetPtr); err != nil && !errors.Is(err, windows.NOERROR) {
		return nil, fmt.Errorf("failed to release received packet: %v", err)
	}

	return packet, nil
}

// ReceivePacket receives a packet from the Wintun session. If no packet is available, waits for a packet to become
// available.
//
// The function will block until a packet is available if the packet queue is empty. If the packet queue is not
// empty, the function will return immediately. If the session handle is invalid or a critical error occurs, the
// function will return an error.
//
// The returned packet is a copy of the internal packet buffer and may be modified or reused immediately.
func (s *Session) ReceivePacket() ([]byte, error) {
	packet, err := s.ReceivePacketNow()
	if err == nil || !errors.Is(err, ErrNoDataAvailable) {
		return packet, err
	}

	if result, _ := windows.WaitForSingleObject(s.waitEvent, windows.INFINITE); result != windows.WAIT_OBJECT_0 {
		return nil, fmt.Errorf("wait event failed: unexpected result %v", result)
	}

	return s.ReceivePacket()
}

// GetRunningDriverVersion determines the version of the Wintun driver currently loaded.
//
// Returns the version in the format "X.Y", where X is the major version and Y is the minor version.
// If the function fails, the return value is an empty string and an error is set.
func (a *Adapter) GetRunningDriverVersion() (string, error) {
	if a.handle == 0 {
		return "", ErrInvalidAdapterHandle
	}

	version, _, err := wintunGetRunningDriverVersionFunc.Call()
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return "", fmt.Errorf("failed to get Wintun driver version: %v", err)
	}

	return fmt.Sprintf("%d.%d", version>>16&0xff, version&0xff), nil
}

// GetAdapterLUID retrieves the LUID of the adapter.
//
// Returns the LUID of the adapter on success, or an error if the
// LUID cannot be retrieved.
func (a *Adapter) GetAdapterLUID() (windows.LUID, error) {
	if a.handle == 0 {
		return windows.LUID{}, ErrInvalidAdapterHandle
	}

	var luid windows.LUID
	if _, _, err := wintunGetAdapterLUIDFunc.Call(a.handle, uintptr(unsafe.Pointer(&luid))); err != nil && !errors.Is(err, windows.NOERROR) {
		return windows.LUID{}, fmt.Errorf("failed to retrieve adapter LUID: %v", err)
	}
	return luid, nil
}

// GetAdapterGUID retrieves the GUID associated with the adapter.
//
// This function first obtains the adapter's LUID and then converts it
// to the corresponding GUID using the Windows API. The GUID is a unique
// identifier for the network adapter.
//
// Returns the GUID of the adapter on success, or an error if the
// LUID cannot be retrieved or converted to a GUID.
func (a *Adapter) GetAdapterGUID() (windows.GUID, error) {
	if a.handle == 0 {
		return windows.GUID{}, ErrInvalidAdapterHandle
	}

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

// GetAdapterIndex retrieves the index of the adapter.
//
// The adapter index is the number used by the Windows API to identify the
// adapter. It is a zero-based index, so the first adapter is 0, the second
// adapter is 1, etc.
//
// The index is used by many of the Windows API functions, such as
// GetAdaptersInfo and GetAdapterAddresses.
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

// GetAdapterName returns the name of the adapter.
//
// The adapter name is the name passed to WintunCreateAdapter or the name of the
// adapter when it was opened with WintunOpenAdapter.
func (a *Adapter) GetAdapterName() string {
	return a.name
}
