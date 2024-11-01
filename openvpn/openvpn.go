//go:build windows

package openvpn

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/XenonCommunity/swiftunnel/swiftypes"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"net"
	"os"
	"syscall"
	"unsafe"
)

const (
	tapDriverKey = `SYSTEM\CurrentControlSet\Control\Class\{4D36E972-E325-11CE-BFC1-08002BE10318}`
	netConfigKey = `SYSTEM\CurrentControlSet\Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}`
)

var (
	errIfceNameNotFound        = errors.New("failed to find the name of interface")
	fileDeviceUnknown          = uint32(0x00000022)
	tapWinIoctlGetMac          = tapControlCode(1, 0)
	tapIoctlSetMediaStatus     = tapControlCode(6, 0)
	tapIoctlConfigTun          = tapControlCode(10, 0)
	iphlpapi                   = syscall.NewLazyDLL("iphlpapi.dll")
	convertInterfaceGuidToLuid = iphlpapi.NewProc("ConvertInterfaceGuidToLuid")
)

//goland:noinspection GoNameStartsWithPackageName
type OpenVPNAdapter struct {
	file *os.File
	name string
	guid swiftypes.GUID
}

func (a *OpenVPNAdapter) Write(buf []byte) (int, error)           { return a.file.Write(buf) }
func (a *OpenVPNAdapter) Read(buf []byte) (int, error)            { return a.file.Read(buf) }
func (a *OpenVPNAdapter) Close() error                            { return a.file.Close() }
func (a *OpenVPNAdapter) GetFD() *os.File                         { return a.file }
func (a *OpenVPNAdapter) GetAdapterName() (string, error)         { return a.name, nil }
func (a *OpenVPNAdapter) GetAdapterGUID() (swiftypes.GUID, error) { return a.guid, nil }

func (a *OpenVPNAdapter) GetAdapterIndex() (uint32, error) {
	ifce, err := net.InterfaceByName(a.name)
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve adapter index: %w", err)
	}
	return uint32(ifce.Index), nil
}

func (a *OpenVPNAdapter) GetAdapterLUID() (swiftypes.LUID, error) {
	var luid swiftypes.LUID
	_, _, err := convertInterfaceGuidToLuid.Call(
		uintptr(unsafe.Pointer(&a.guid)),
		uintptr(unsafe.Pointer(&luid)),
	)
	if err != nil && !errors.Is(err, windows.NOERROR) {
		return swiftypes.LUID{}, fmt.Errorf("failed to get adapter LUID: %w", err)
	}
	return luid, nil
}

func NewOpenVPNAdapter(componentID swiftypes.GUID, name string, localIP net.IP, remoteNet *net.IPNet, tun bool) (*OpenVPNAdapter, error) {
	deviceID, err := getDeviceID(name, componentID)
	if err != nil {
		return nil, err
	}
	path := `\\.\Global\` + deviceID + `.tap`

	fd, err := openTapDevice(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = fd.Close()
		}
	}()

	mac, err := getMacAddress(fd)
	if err != nil {
		return nil, err
	}

	adapter := &OpenVPNAdapter{file: fd, guid: componentID}
	if err := setStatus(fd, true); err != nil {
		return nil, err
	}
	if tun {
		if err := setTUN(fd, localIP, remoteNet); err != nil {
			return nil, err
		}
	}
	adapter.name, err = findInterfaceName(mac)
	return adapter, err
}

func openTapDevice(path string) (*os.File, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	file, err := syscall.CreateFile(pathPtr, syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_SYSTEM|syscall.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(file), path), nil
}

func getDeviceID(name string, componentID swiftypes.GUID) (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, tapDriverKey, registry.READ)
	if err != nil {
		return "", fmt.Errorf("could not access TAP driver registry key: %w", err)
	}
	defer key.Close()

	keys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return "", err
	}

	for _, subKey := range keys {
		if id, err := findDeviceByID(subKey, name, componentID); err == nil {
			return id, nil
		}
	}
	return "", fmt.Errorf("device with ComponentId '%s' and InterfaceName '%s' not found", componentID, name)
}

func findDeviceByID(subKey, name string, componentID swiftypes.GUID) (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, tapDriverKey+`\`+subKey, registry.READ)
	if err != nil {
		return "", err
	}
	defer key.Close()

	val, _, err := key.GetStringValue("ComponentId")
	if err != nil || val != componentID.String() {
		return "", err
	}

	netCfgID, _, err := key.GetStringValue("NetCfgInstanceId")
	if err != nil {
		return "", err
	}

	if len(name) > 0 {
		connKey := fmt.Sprintf("%s\\%s\\Connection", netConfigKey, netCfgID)
		conn, err := registry.OpenKey(registry.LOCAL_MACHINE, connKey, registry.READ)
		if err != nil {
			return "", err
		}
		defer conn.Close()

		if value, _, err := conn.GetStringValue("Name"); err != nil || value != name {
			return "", fmt.Errorf("interface value '%s' does not match", name)
		}
	}
	return netCfgID, nil
}

func getMacAddress(fd *os.File) ([]byte, error) {
	mac := make([]byte, 6)
	var bytesReturned uint32
	err := windows.DeviceIoControl(windows.Handle(fd.Fd()), tapWinIoctlGetMac, &mac[0], uint32(len(mac)), &mac[0], uint32(len(mac)), &bytesReturned, nil)
	if err != nil {
		return nil, err
	}
	return mac, nil
}

func setStatus(fd *os.File, status bool) error {
	code := []byte{0x00, 0x00, 0x00, 0x00}
	if status {
		code[0] = 0x01
	}
	var bytesReturned uint32
	return windows.DeviceIoControl(windows.Handle(fd.Fd()), tapIoctlSetMediaStatus, &code[0], 4, nil, 0, &bytesReturned, nil)
}

func setTUN(fd *os.File, localIP net.IP, remoteNet *net.IPNet) error {
	if localIP.To4() == nil {
		return fmt.Errorf("invalid IPv4 address: %s", localIP)
	}
	data := append(localIP.To4(), remoteNet.IP.To4()...)
	data = append(data, remoteNet.Mask...)
	var bytesReturned uint32
	return windows.DeviceIoControl(windows.Handle(fd.Fd()), tapIoctlConfigTun, &data[0], 12, nil, 0, &bytesReturned, nil)
}

func findInterfaceName(mac []byte) (string, error) {
	ifces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifce := range ifces {
		if len(ifce.HardwareAddr) >= 6 && bytes.Equal(ifce.HardwareAddr[:6], mac) {
			return ifce.Name, nil
		}
	}
	return "", errIfceNameNotFound
}

func ctlCode(deviceType, function, method, access uint32) uint32 {
	return (deviceType << 16) | (access << 14) | (function << 2) | method
}

func tapControlCode(request, method uint32) uint32 {
	return ctlCode(fileDeviceUnknown, request, method, 0)
}
