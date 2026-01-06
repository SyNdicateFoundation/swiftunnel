//go:build windows

package wintun

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
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
	dllInitOnce sync.Once
	dllInitErr  error
)

var (
	ErrInvalidAdapterHandle = errors.New("adapter handle is invalid or closed")
	ErrInvalidSessionHandle = errors.New("session handle is invalid or closed")
	ErrEmptyPacket          = errors.New("packet cannot be empty")
	ErrNoDataAvailable      = errors.New("no more data is available")
	ErrBufferTooSmall       = errors.New("destination buffer is too small")
)

// ensureDLL extracts the embedded Wintun DLL to a cache directory and loads it.
func ensureDLL() error {
	dllInitOnce.Do(func() {
		hash := sha256.Sum256(wintunDLLData)
		hexHash := hex.EncodeToString(hash[:])

		localAppData, err := os.UserCacheDir()
		if err != nil {
			dllInitErr = err
			return
		}

		dirPath := filepath.Join(localAppData, "swiftunnel", "driver")
		dllPath := filepath.Join(dirPath, "wintun.dll")

		valid := false
		if f, err := os.Open(dllPath); err == nil {
			h := sha256.New()
			if _, err := io.Copy(h, f); err == nil {
				if hex.EncodeToString(h.Sum(nil)) == hexHash {
					valid = true
				}
			}
			f.Close()
		}

		if !valid {
			os.Remove(dllPath)

			if err := os.MkdirAll(dirPath, 0755); err != nil {
				dllInitErr = fmt.Errorf("failed to create dir: %w", err)
				return
			}

			if err := os.WriteFile(dllPath, wintunDLLData, 0600); err != nil {
				dllInitErr = fmt.Errorf("failed to write dll: %w", err)
				return
			}
		}

		wintunDLL, err = windows.LoadDLL(dllPath)
		if err != nil {
			dllInitErr = err
			return
		}

		dllInitErr = loadWintunFunctions()

		runtime.SetFinalizer(wintunDLL, func(dll *windows.DLL) {
			UninstallWintun()
		})
	})
	return dllInitErr
}

// loadWintunFunctions maps exported symbols from wintun.dll to internal Go variables.
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
			return fmt.Errorf("failed to find procedure %s", name)
		}
	}

	return nil
}

func init() {
	if err := ensureDLL(); err != nil {
		panic(err)
	}
}

// WintunAdapter represents a handle to a Wintun network interface.
type WintunAdapter struct {
	name       string
	tunnelType string
	handle     uintptr
	closed     atomic.Bool
}

// WintunSession represents an active high-performance ring-buffer session on a Wintun adapter.
type WintunSession struct {
	adapter   *WintunAdapter
	handle    uintptr
	waitEvent windows.Handle
	closed    atomic.Bool
}

// NewWintunAdapter creates a new adapter with a default GUID.
func NewWintunAdapter(name, tunnelType string) (*WintunAdapter, error) {
	return NewWintunAdapterWithGUID(name, tunnelType, swiftypes.NilGUID)
}

// NewWintunAdapterWithGUID initializes a Wintun adapter with a specific name and GUID.
func NewWintunAdapterWithGUID(name, tunnelType string, componentID swiftypes.GUID) (*WintunAdapter, error) {
	if err := ensureDLL(); err != nil {
		return nil, err
	}

	if name == "" {
		name = "Wintun"
	}

	if tunnelType == "" {
		tunnelType = "VPN"
	}

	namePtr, _ := windows.UTF16PtrFromString(name)
	tunnelTypePtr, _ := windows.UTF16PtrFromString(tunnelType)

	var handle uintptr
	var err error

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
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	adapter := &WintunAdapter{
		name:       name,
		tunnelType: tunnelType,
		handle:     handle,
	}

	runtime.SetFinalizer(adapter, func(a *WintunAdapter) {
		_ = a.Close()
	})

	return adapter, nil
}

// Close terminates the Wintun adapter and releases the driver handle.
func (a *WintunAdapter) Close() error {
	if a.closed.Swap(true) {
		return nil
	}

	runtime.SetFinalizer(a, nil)

	if a.handle == 0 {
		return ErrInvalidAdapterHandle
	}

	if _, _, err := wintunCloseAdapterFunc.Call(a.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to close Wintun adapter: %v", err)
	}

	a.handle = 0

	return nil
}

// StartSession initializes the ring buffer for the adapter.
func (a *WintunAdapter) StartSession(capacity uint32) (*WintunSession, error) {
	if a.closed.Load() || a.handle == 0 {
		return nil, ErrInvalidAdapterHandle
	}

	handle, _, err := wintunStartSessionFunc.Call(a.handle, uintptr(capacity))
	if handle == 0 {
		return nil, fmt.Errorf("failed to start session: %v", err)
	}

	waitEvent, _, err := wintunGetReadWaitEventFunc.Call(handle)
	if err != nil && !errors.Is(err, windows.NOERROR) {
		wintunEndSessionFunc.Call(handle)
		return nil, fmt.Errorf("failed to get wait event: %v", err)
	}

	session := &WintunSession{
		adapter:   a,
		handle:    handle,
		waitEvent: windows.Handle(waitEvent),
	}

	runtime.SetFinalizer(session, func(s *WintunSession) {
		_ = s.Close()
	})

	return session, nil
}

// Close terminates the Wintun session and ends the ring buffer communication.
func (s *WintunSession) Close() error {
	if s.closed.Swap(true) {
		return nil
	}

	runtime.SetFinalizer(s, nil)

	if s.handle == 0 {
		return ErrInvalidSessionHandle
	}

	if _, _, err := wintunEndSessionFunc.Call(s.handle); err != nil && !errors.Is(err, windows.NOERROR) {
		return fmt.Errorf("failed to end session: %v", err)
	}

	s.handle = 0

	return s.adapter.Close()
}

// Write allocates space in the ring buffer and copies the packet for transmission.
//
//goland:noinspection GoVetUnsafePointer
func (s *WintunSession) Write(buf []byte) (int, error) {
	if s.closed.Load() {
		return 0, ErrInvalidSessionHandle
	}
	if len(buf) == 0 {
		return 0, ErrEmptyPacket
	}

	dataPtr, _, err := wintunAllocateSendPacketFunc.Call(s.handle, uintptr(len(buf)))
	if (err != nil && !errors.Is(err, windows.NOERROR)) || dataPtr == 0 {
		return 0, fmt.Errorf("ring buffer full or error: %v", err)
	}

	//goland:noinspection GoRedundantConversion
	dst := unsafe.Slice((*byte)(unsafe.Pointer(dataPtr)), len(buf))
	copy(dst, buf)

	if _, _, err := wintunSendPacketFunc.Call(s.handle, dataPtr, uintptr(len(buf))); err != nil && !errors.Is(err, windows.NOERROR) {
		return 0, fmt.Errorf("failed to send packet: %v", err)
	}

	return len(buf), nil
}

// ReadNow attempts a non-blocking read from the Wintun ring buffer.
func (s *WintunSession) ReadNow(buf []byte) (int, error) {
	if s.closed.Load() {
		return 0, ErrInvalidSessionHandle
	}

	var packetSize uint32

	packetPtr, _, err := wintunReceivePacketFunc.Call(s.handle, uintptr(unsafe.Pointer(&packetSize)))
	if packetPtr == 0 {
		if err != nil && err.Error() == "No more data is available." {
			return 0, ErrNoDataAvailable
		}

		if !errors.Is(err, windows.NOERROR) {
			return 0, err
		}

		return 0, ErrNoDataAvailable
	}

	defer wintunReleaseReceivePacketFunc.Call(s.handle, packetPtr)

	if int(packetSize) > len(buf) {
		return 0, ErrBufferTooSmall
	}

	//goland:noinspection GoRedundantConversion,GoVetUnsafePointer
	src := unsafe.Slice((*byte)(unsafe.Pointer(packetPtr)), packetSize)
	copy(buf, src)

	return int(packetSize), nil
}

// Read performs a blocking read from the Wintun session, waiting on the driver event.
func (s *WintunSession) Read(buf []byte) (int, error) {
	for {
		if s.closed.Load() {
			return 0, ErrInvalidSessionHandle
		}

		n, err := s.ReadNow(buf)
		if err == nil {
			return n, nil
		}
		if !errors.Is(err, ErrNoDataAvailable) {
			return 0, err
		}

		event, _ := windows.WaitForSingleObject(s.waitEvent, 100)

		if event == uint32(windows.WAIT_TIMEOUT) {
			continue
		}

		if event == windows.WAIT_FAILED {
			return 0, errors.New("wait failed")
		}
	}
}

// GetFD returns nil as Wintun uses a custom ring buffer rather than a standard file descriptor.
func (s *WintunSession) GetFD() *os.File {
	return s.adapter.GetFD()
}

// GetAdapterName retrieves the adapter name from the parent structure.
func (s *WintunSession) GetAdapterName() (string, error) {
	return s.adapter.GetAdapterName()
}

// GetAdapterIndex retrieves the interface index via the parent adapter.
func (s *WintunSession) GetAdapterIndex() (int, error) {
	return s.adapter.GetAdapterIndex()
}

// GetAdapterLUID retrieves the Windows LUID via the parent adapter.
func (s *WintunSession) GetAdapterLUID() (swiftypes.LUID, error) {
	return s.adapter.GetAdapterLUID()
}

// GetAdapterGUID retrieves the Windows GUID via the parent adapter.
func (s *WintunSession) GetAdapterGUID() (swiftypes.GUID, error) {
	return s.adapter.GetAdapterGUID()
}

// GetFD returns nil for the adapter.
func (a *WintunAdapter) GetFD() *os.File {
	return nil
}

// GetAdapterName returns the friendly name of the adapter.
func (a *WintunAdapter) GetAdapterName() (string, error) {
	return a.name, nil
}

// GetAdapterLUID uses the Wintun API to fetch the 64-bit LUID.
func (a *WintunAdapter) GetAdapterLUID() (swiftypes.LUID, error) {
	if a.handle == 0 {
		return swiftypes.NilLUID, ErrInvalidAdapterHandle
	}

	var luid64 uint64

	_, _, err := wintunGetAdapterLUIDFunc.Call(a.handle, uintptr(unsafe.Pointer(&luid64)))
	if !errors.Is(err, windows.NOERROR) {
		return swiftypes.NilLUID, fmt.Errorf("failed to get LUID: %v", err)
	}

	return swiftypes.NewLUID(luid64), nil
}

// GetAdapterGUID converts the adapter LUID to a GUID using IP Helper.
func (a *WintunAdapter) GetAdapterGUID() (swiftypes.GUID, error) {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return swiftypes.NilGUID, err
	}

	var guid swiftypes.GUID
	_, _, err = convertInterfaceLuidToGuid.Call(
		uintptr(unsafe.Pointer(&luid)),
		uintptr(unsafe.Pointer(&guid)),
	)

	if err != nil && !errors.Is(err, windows.NOERROR) {
		return swiftypes.NilGUID, fmt.Errorf("ConvertInterfaceLuidToGuid failed: %w", err)
	}

	return guid, nil
}

// GetAdapterIndex converts the adapter LUID to an IP IF index.
func (a *WintunAdapter) GetAdapterIndex() (int, error) {
	luid, err := a.GetAdapterLUID()
	if err != nil {
		return 0, err
	}

	var index uint32
	ret, _, _ := convertLUIDToIndex.Call(
		uintptr(unsafe.Pointer(&luid)),
		uintptr(unsafe.Pointer(&index)),
	)

	if ret != 0 {
		return 0, fmt.Errorf("ConvertInterfaceLuidToIndex failed: %d", ret)
	}

	return int(index), nil
}

// GetRunningDriverVersion retrieves the version string from the loaded driver.
func (a *WintunAdapter) GetRunningDriverVersion() (string, error) {
	if err := ensureDLL(); err != nil {
		return "", err
	}

	v, _, _ := wintunGetRunningDriverVersionFunc.Call()
	if v == 0 {
		return "", errors.New("failed to get driver version")
	}

	return fmt.Sprintf("%d.%d", (v>>16)&0xFF, v&0xFF), nil
}

// UninstallWintun attempts to remove the Wintun driver from the system.
func UninstallWintun() {
	if wintunDeleteDriverFunc != nil {
		wintunDeleteDriverFunc.Call()
	}

	runtime.SetFinalizer(wintunDLL, nil)
}
