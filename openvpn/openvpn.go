//go:build windows

package openvpn

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"net"
	"os"
	"unsafe"
)

const (
	tapDriverKey = `SYSTEM\CurrentControlSet\Control\Class\{4D36E972-E325-11CE-BFC1-08002BE10318}`
	netConfigKey = `SYSTEM\CurrentControlSet\Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}`
)

var (
	tapWinIoctlGetMac      = tapControlCode(1, 0)
	tapIoctlSetMediaStatus = tapControlCode(6, 0)
	tapIoctlConfigTun      = tapControlCode(10, 0)
)

var (
	iphlpapi                   = windows.NewLazySystemDLL("iphlpapi.dll")
	convertInterfaceGuidToLuid = iphlpapi.NewProc("ConvertInterfaceGuidToLuid")
)

// OpenVPNAdapter represents a Windows TAP-Windows adapter session.
type OpenVPNAdapter struct {
	file *os.File
	name string
	guid swiftypes.GUID
}

// Write transmits data to the TAP device.
func (a *OpenVPNAdapter) Write(buf []byte) (int, error) {
	return a.file.Write(buf)
}

// Read receives data from the TAP device.
func (a *OpenVPNAdapter) Read(buf []byte) (int, error) {
	return a.file.Read(buf)
}

// Close releases the TAP device handle.
func (a *OpenVPNAdapter) Close() error {
	return a.file.Close()
}

// GetFD returns the OS file object for the TAP handle.
func (a *OpenVPNAdapter) GetFD() *os.File {
	return a.file
}

// GetAdapterName returns the friendly name of the TAP interface.
func (a *OpenVPNAdapter) GetAdapterName() (string, error) {
	return a.name, nil
}

// GetAdapterGUID returns the GUID associated with the TAP interface.
func (a *OpenVPNAdapter) GetAdapterGUID() (swiftypes.GUID, error) {
	return a.guid, nil
}

// GetAdapterIndex returns the interface index for the TAP device.
func (a *OpenVPNAdapter) GetAdapterIndex() (int, error) {
	ifce, err := net.InterfaceByName(a.name)
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve adapter index: %w", err)
	}
	return ifce.Index, nil
}

// GetAdapterLUID converts the interface GUID to a Windows LUID.
func (a *OpenVPNAdapter) GetAdapterLUID() (swiftypes.LUID, error) {
	var luid swiftypes.LUID

	ret, _, _ := convertInterfaceGuidToLuid.Call(
		uintptr(unsafe.Pointer(&a.guid)),
		uintptr(unsafe.Pointer(&luid)),
	)

	if ret != 0 {
		return swiftypes.NilLUID, fmt.Errorf("ConvertInterfaceGuidToLuid failed: %d", ret)
	}

	return luid, nil
}

// NewOpenVPNAdapter opens and configures a TAP-Windows device.
func NewOpenVPNAdapter(adapterGUID swiftypes.GUID, name string, localIP net.IP, remoteNet *net.IPNet, tun bool) (*OpenVPNAdapter, error) {
	deviceID, err := findDeviceID(name, adapterGUID)
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
		return nil, fmt.Errorf("failed to get MAC from TAP device: %w", err)
	}

	actualName, err := findInterfaceNameByMAC(mac)
	if err != nil {
		actualName = name
	}

	if err := setStatus(fd, true); err != nil {
		return nil, fmt.Errorf("failed to set TAP status to up: %w", err)
	}

	if tun {
		if err := setTUN(fd, localIP, remoteNet); err != nil {
			return nil, fmt.Errorf("failed to configure TUN IP: %w", err)
		}
	}

	parsedGUID, err := swiftypes.ParseGUID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid GUID from registry: %w", err)
	}

	return &OpenVPNAdapter{
		file: fd,
		name: actualName,
		guid: parsedGUID,
	}, nil
}

// openTapDevice creates a file handle to the TAP device path.
func openTapDevice(path string) (*os.File, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_SYSTEM|windows.FILE_FLAG_OVERLAPPED,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAP device at %s: %w", path, err)
	}

	return os.NewFile(uintptr(handle), path), nil
}

// findDeviceID searches the Windows registry for a matching TAP adapter.
func findDeviceID(targetName string, targetGUID swiftypes.GUID) (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, tapDriverKey, registry.READ)
	if err != nil {
		return "", fmt.Errorf("failed to access TAP driver registry key: %w", err)
	}
	defer k.Close()

	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return "", err
	}

	for _, subKeyName := range subkeys {
		subKeyPath := tapDriverKey + `\` + subKeyName
		subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, subKeyPath, registry.READ)
		if err != nil {
			continue
		}

		componentId, _, _ := subKey.GetStringValue("ComponentId")
		if componentId != "tap0901" {
			subKey.Close()
			continue
		}

		netCfgID, _, err := subKey.GetStringValue("NetCfgInstanceId")
		subKey.Close()
		if err != nil {
			continue
		}

		if targetGUID != swiftypes.NilGUID {
			if netCfgID == targetGUID.String() {
				return netCfgID, nil
			}
			continue
		}

		if targetName != "" {
			connKeyPath := fmt.Sprintf(`%s\%s\Connection`, netConfigKey, netCfgID)

			connKey, err := registry.OpenKey(registry.LOCAL_MACHINE, connKeyPath, registry.READ)
			if err != nil {
				continue
			}

			nameValue, _, err := connKey.GetStringValue("Name")
			connKey.Close()

			if err == nil && nameValue == targetName {
				return netCfgID, nil
			}
		}
	}

	return "", fmt.Errorf("TAP adapter not found (Name: %s, GUID: %s)", targetName, targetGUID)
}

// getMacAddress retrieves the hardware address of the TAP device via IOCTL.
func getMacAddress(f *os.File) ([]byte, error) {
	mac := make([]byte, 6)

	var returned uint32

	err := windows.DeviceIoControl(
		windows.Handle(f.Fd()),
		tapWinIoctlGetMac,
		&mac[0],
		uint32(len(mac)),
		&mac[0],
		uint32(len(mac)),
		&returned,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return mac, nil
}

// setStatus sets the TAP device media status to connected or disconnected.
func setStatus(f *os.File, status bool) error {
	code := []byte{0x00, 0x00, 0x00, 0x00}

	if status {
		code[0] = 0x01
	}

	var returned uint32

	return windows.DeviceIoControl(
		windows.Handle(f.Fd()),
		tapIoctlSetMediaStatus,
		&code[0],
		uint32(len(code)),
		nil,
		0,
		&returned,
		nil,
	)
}

// setTUN configures the IPv4 tunnel parameters for the TAP driver.
func setTUN(f *os.File, localIP net.IP, remoteNet *net.IPNet) error {
	ipv4 := localIP.To4()
	if ipv4 == nil {
		return errors.New("TAP driver currently only supports IPv4 for TUN config")
	}

	data := make([]byte, 0, 12)
	data = append(data, ipv4...)
	data = append(data, remoteNet.IP.To4()...)
	data = append(data, remoteNet.Mask...)

	if len(data) != 12 {
		return fmt.Errorf("invalid TUN config data length: %d", len(data))
	}

	var returned uint32
	return windows.DeviceIoControl(
		windows.Handle(f.Fd()),
		tapIoctlConfigTun,
		&data[0],
		uint32(len(data)),
		nil,
		0,
		&returned,
		nil,
	)
}

// findInterfaceNameByMAC matches a MAC address to a network interface name.
func findInterfaceNameByMAC(mac []byte) (string, error) {
	ifces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, ifce := range ifces {
		if len(ifce.HardwareAddr) >= 6 && bytes.Equal(ifce.HardwareAddr[:6], mac) {
			return ifce.Name, nil
		}
	}

	return "", errors.New("interface name not found for MAC")
}

// tapControlCode calculates the Windows DeviceIoControl code for the TAP driver.
func tapControlCode(request, method uint32) uint32 {
	return (0x00000022 << 16) | (0 << 14) | (request << 2) | method
}
