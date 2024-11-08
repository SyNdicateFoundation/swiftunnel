//go:build windows

package wintun

import (
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"os"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	wintunDLL                         *windows.DLL
	wintunCreateAdapterFunc           *windows.Proc
	wintunCloseAdapterFunc            *windows.Proc
	wintunDeleteDriverFunc            *windows.Proc
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
	iphlpapi                   = windows.NewLazySystemDLL("iphlpapi.dll")
	convertLUIDToIndex         = iphlpapi.NewProc("ConvertInterfaceLuidToIndex")
	convertInterfaceLuidToGuid = iphlpapi.NewProc("ConvertInterfaceLuidToGuid")
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

		runtime.SetFinalizer(wintunDLL, func(dll *windows.DLL) {
			UninstallWintun()
		})
	})
}

func loadWintunDLL() error {
	dllFile, err := os.CreateTemp("", "wintun_*.dll")
	if err != nil {
		return fmt.Errorf("failed to create temp file for Wintun DLL: %w", err)
	}

	defer func(name string) {
		_ = os.Remove(name)
	}(dllFile.Name())

	if _, err := dllFile.Write(wintunDLLData); err != nil {
		return fmt.Errorf("failed to write Wintun DLL data: %w", err)
	}
	if err := dllFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	wintunDLL = windows.MustLoadDLL(dllFile.Name())
	return loadWintunFunctions()
}

func loadWintunFunctions() error {
	procs := map[string]**windows.Proc{
		"WintunCreateAdapter":           &wintunCreateAdapterFunc,
		"WintunDeleteDriver":            &wintunDeleteDriverFunc,
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

//goland:noinspection GoNameStartsWithPackageName
type WintunAdapter struct {
	name       string
	tunnelType string
	handle     uintptr
}

//goland:noinspection GoNameStartsWithPackageName
type WintunSession struct {
	*WintunAdapter
	handle    uintptr
	waitEvent windows.Handle
}

func NewWintunAdapter(name, tunnelType string) (*WintunAdapter, error) {
	return NewWintunAdapterWithGUID(name, tunnelType, swiftypes.NilGUID)
}

func NewWintunAdapterWithGUID(name, tunnelType string, componentID swiftypes.GUID) (*WintunAdapter, error) {
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
	if componentID != swiftypes.NilGUID {
		handle, _, err = wintunCreateAdapterFunc.Call(
			uintptr(unsafe.Pointer(namePtr)),
			uintptr(unsafe.Pointer(tunnelTypePtr)),
			uintptr(unsafe.Pointer(&componentID)),
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

	a := &WintunAdapter{
		name:       name,
		tunnelType: tunnelType,
		handle:     handle,
	}

	runtime.SetFinalizer(a, func(a *WintunAdapter) {
		_ = a.Close()
	})

	return a, nil
}

func (a *WintunAdapter) Close() error {
	if a.handle == 0 {
		return ErrInvalidAdapterHandle
	}

	if _, _, err := wintunCloseAdapterFunc.Call(a.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to close Wintun adapter: %v", err)
	}

	runtime.SetFinalizer(a, nil)

	return nil
}

func (a *WintunAdapter) StartSession(capacity uint32) (*WintunSession, error) {
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

	s := &WintunSession{handle: handle, waitEvent: windows.Handle(waitEvent), WintunAdapter: a}

	runtime.SetFinalizer(s, func(s *WintunSession) {
		_ = s.Close()
	})

	return s, nil
}

func (s *WintunSession) Close() error {
	defer runtime.SetFinalizer(s, nil)

	if s.handle == 0 {
		return ErrInvalidSessionHandle
	}

	if _, _, err := wintunEndSessionFunc.Call(s.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to end Wintun session: %v", err)
	}

	return s.WintunAdapter.Close()
}

func (s *WintunSession) Write(buf []byte) (int, error) {
	if s.handle == 0 {
		return 0, ErrInvalidSessionHandle
	}

	if len(buf) == 0 {
		return 0, ErrEmptyPacket
	}

	dataPtr, _, err := wintunAllocateSendPacketFunc.Call(s.handle, uintptr(len(buf)))
	if (err != nil && !errors.Is(err, windows.NOERROR)) || dataPtr == 0 {
		return 0, fmt.Errorf("failed to allocate Wintun buf: %v", err)
	}

	//goland:noinspection GoRedundantConversion
	copy(unsafe.Slice((*byte)(unsafe.Pointer(dataPtr)), len(buf)), buf)

	if _, _, err := wintunSendPacketFunc.Call(s.handle, dataPtr, uintptr(len(buf))); err != nil && !errors.Is(err, windows.NOERROR) {
		return 0, fmt.Errorf("failed to send Wintun buf: %v", err)
	}

	return len(buf), nil
}

func (s *WintunSession) ReadNow(buf []byte) (int, error) {
	if s.handle == 0 {
		return 0, ErrInvalidSessionHandle
	}

	var packetSize uint32
	packetPtr, _, err := wintunReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetSize)))
	if packetPtr == 0 || !errors.Is(err, windows.NOERROR) {
		if err != nil {
			if err.Error() == "No more data is available." {
				return 0, ErrNoDataAvailable
			}
		} else {
			return 0, ErrFailedToReceive
		}
	}

	//goland:noinspection GoRedundantConversion
	n := copy(buf, unsafe.Slice((*byte)(unsafe.Pointer(packetPtr)), packetSize))

	if _, _, err := wintunReleaseReceivePacketFunc.Call(s.handle, packetPtr); err != nil && !errors.Is(err, windows.NOERROR) {
		return 0, fmt.Errorf("failed to release received packet: %v", err)
	}

	return n, nil
}

func (s *WintunSession) Read(buf []byte) (int, error) {
	n, err := s.ReadNow(buf)
	if err == nil || !errors.Is(err, ErrNoDataAvailable) {
		return n, err
	}

	if result, _ := windows.WaitForSingleObject(s.waitEvent, windows.INFINITE); result != windows.WAIT_OBJECT_0 {
		return 0, fmt.Errorf("wait event failed: unexpected result %v", result)
	}

	return s.Read(buf)
}

func (a *WintunAdapter) GetRunningDriverVersion() (string, error) {
	if a.handle == 0 {
		return "", ErrInvalidAdapterHandle
	}

	version, _, err := wintunGetRunningDriverVersionFunc.Call()
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return "", fmt.Errorf("failed to get Wintun driver version: %v", err)
	}

	return fmt.Sprintf("%d.%d", version>>16&0xff, version&0xff), nil
}

func (a *WintunAdapter) GetFD() *os.File {
	return nil
}

func (a *WintunAdapter) GetAdapterName() (string, error) {
	return a.name, nil
}

func (a *WintunAdapter) GetAdapterLUID() (swiftypes.LUID, error) {
	if a.handle == 0 {
		return swiftypes.NilLUID, ErrInvalidAdapterHandle
	}

	var luid64 uint64
	if _, _, err := wintunGetAdapterLUIDFunc.Call(a.handle, uintptr(unsafe.Pointer(&luid64))); err != nil && !errors.Is(err, windows.NOERROR) {
		return swiftypes.NilLUID, fmt.Errorf("failed to retrieve adapter LUID: %v", err)
	}

	return swiftypes.NewLUID(luid64), nil
}

func (a *WintunAdapter) GetAdapterGUID() (swiftypes.GUID, error) {
	if a.handle == 0 {
		return swiftypes.NilGUID, ErrInvalidAdapterHandle
	}

	luid, err := a.GetAdapterLUID()
	if err != nil {
		return swiftypes.NilGUID, err
	}

	var retrievedGUID swiftypes.GUID

	if _, _, err := convertInterfaceLuidToGuid.Call(uintptr(unsafe.Pointer(&luid)), uintptr(unsafe.Pointer(&retrievedGUID))); err != nil && !errors.Is(err, windows.NOERROR) {
		return swiftypes.NilGUID, fmt.Errorf("failed to convert LUID to GUID: %v", err)
	}

	return retrievedGUID, nil
}

func (a *WintunAdapter) GetAdapterIndex() (int, error) {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return 0, err
	}

	var index uint32
	if _, _, err := convertLUIDToIndex.Call(uintptr(unsafe.Pointer(&luid)), uintptr(unsafe.Pointer(&index))); err != nil && !errors.Is(err, windows.NOERROR) {
		return 0, fmt.Errorf("failed to convert LUID to index: %v", err)
	}

	return int(index), nil
}

func UninstallWintun() {
	if _, _, err := wintunDeleteDriverFunc.Call(); err != nil {
		return
	}

	runtime.SetFinalizer(wintunDLL, nil)
}
