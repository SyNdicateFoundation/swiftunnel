# Swiftunnel

Swiftunnel is a high-performance, cross-platform Go library designed to create and manage virtual network interfaces (
TUN/TAP). It abstracts the complexities of platform-specific networking APIs into a unified, clean interface for
building VPNs, overlay networks, and custom protocol tunnels.

## Purpose

The primary goal of Swiftunnel is to provide a consistent programming model for network tunneling across Windows, Linux,
and macOS. It handles the low-level system calls, driver management, and interface configuration (IP, MTU, DNS) so
developers can focus on packet processing logic.

## Key Features

* **Cross-Platform Support**: Unified API for Windows, Linux, and macOS.
* **Wintun Integration**: Built-in support for the high-performance Wintun driver on Windows, including automatic driver
  extraction and loading.
* **Multi-Driver Architecture**: Supports both Wintun (Layer 3) and OpenVPN TAP-Windows (Layer 2/3) drivers on Windows.
* **Native Linux Support**: Direct integration with the Linux TUN/TAP subsystem via standard ioctl calls.
* **Dual-Stack macOS Support**: Supports both the native `utun` system driver and third-party TunTapOSX drivers.
* **Automatic Configuration**: Simplifies the assignment of unicast IP addresses, CIDR masks, and MTU settings.
* **DNS Management**: Provides native APIs to configure interface-specific DNS servers and search domains.
* **Functional Configuration**: Utilizes a clean functional-options pattern for flexible and readable initialization.

---

## Technical Documentation

### Core Components

#### 1. `swiftconfig.Config`

The configuration object used to define how the interface should be created. It varies slightly by platform but shares a
common functional initialization pattern.

#### 2. `SwiftInterface`

The primary object representing the virtual tunnel. It implements `io.ReadWriteCloser`, allowing you to use standard Go
patterns to move network packets.

#### 3. `swiftutils`

A collection of helper functions for:

* **Packet Analysis**: Validating and extracting source/destination addresses from IPv4 and IPv6 packets.
* **System DNS**: Purging the system DNS resolver cache using native APIs (Windows) or system controllers (Unix).

---

## Installation

```bash
go get github.com/SyNdicateFoundation/swiftunnel
```

---

## Full Usage Example

The following example demonstrates how to initialize a TUN interface, configure its networking parameters, and enter a
basic packet processing loop.

```go
package main

import (
	"fmt"
	"github.com/SyNdicateFoundation/swiftunnel"
	"github.com/SyNdicateFoundation/swiftunnel/swiftconfig"
	"github.com/SyNdicateFoundation/swiftunnel/swiftutils"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"log"
)

func main() {
	// 1. Define configuration using functional options
	cfg, err := swiftconfig.New(
		swiftconfig.WithAdapterName("SwiftunnelNode"),
		swiftconfig.WithUnicastIP("10.8.0.2/24"),
		swiftconfig.WithMTU(1400),
	)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	
	// 2. Initialize the interface
	// On Windows, this will automatically extract and load Wintun.dll
	adapter, err := swiftunnel.NewSwiftInterface(cfg)
	if err != nil {
		log.Fatalf("Failed to create interface: %v", err)
	}
	defer adapter.Close()
	
	// 3. Set the interface to UP
	if err := adapter.SetStatus(swiftypes.InterfaceUp); err != nil {
		log.Fatalf("Failed to activate interface: %v", err)
	}
	
	fmt.Printf("Interface %s is now active.\n", cfg.AdapterName)
	
	// 4. Packet Processing Loop
	packet := make([]byte, 2048)
	for {
		n, err := adapter.Read(packet)
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}
		
		// Use swiftutils to inspect the packet
		if swiftutils.IsIPv4(packet[:n]) {
			src := swiftutils.IPv4Source(packet[:n])
			dst := swiftutils.IPv4Destination(packet[:n])
			fmt.Printf("IPv4 Packet: %s -> %s (%d bytes)\n", src, dst, n)
		}
		
		// Logic to forward packet to a remote peer would go here
		// _, err = adapter.Write(packet[:n])
	}
}
```

---

## Platform-Specific Implementation Details

### Windows

* **Wintun**: Highly recommended for performance. Requires no external installation as the DLL is embedded in the
  binary.
* **TAP-Windows**: Useful for Layer 2 (TAP) emulation. Requires the OpenVPN TAP driver to be pre-installed on the
  system.

### Linux

* Requires `CAP_NET_ADMIN` privileges to create and configure interfaces.
* Supports both persistent and non-persistent interfaces.

### macOS

* **System Driver**: Uses the native `utun` control socket. Fast and requires no third-party extensions.
* **TunTapOSX**: Required only if TAP (Layer 2) support is needed. Requires kernel extensions.

---

## Contribution Guidelines

Contributions to the Swiftunnel project are welcome. Please ensure that all new code follows the project's
performance-oriented architecture and provides cross-platform compatibility where applicable. For major changes, please
open an issue first to discuss the proposed updates.

---

## License

Copyright (c) 2023-2026 SyNdicateFoundation. All rights reserved. Use of this source code is governed by the proprietary
license found in the LICENSE file.