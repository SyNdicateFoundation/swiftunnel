# 🎉 wintungo: A Go Wrapper for Wintun.NET 🚀

Welcome to **wintungo**, a Golang wrapper for the [wintun](https://www.wintun.net/) project! Wintun is a fast and
efficient tunnel interface for Windows, and this package provides an easy way to use its functionalities in your Go
applications.

## 📦 Features

- Create and manage Wintun adapters with ease.
- Start and end sessions on Wintun adapters.
- Send and receive packets effortlessly.
- Retrieve the running version of the Wintun driver.
- Works seamlessly with the netlink package for network management.

## ⚙️ Installation

To install the wintungo package, use the following command:

```bash
go get github.com/XenonCommunity/wintungo
```

## 🛠️ Usage

Here's a quick example to get you started:

```go
package main

import (
	"fmt"
	"log"
	"github.com/XenonCommunity/wintungo"
)

func main() {
	
	adapter, err := wintun.NewWintunAdapter("MyWintunAdapter", "MyTunnelType")
	if err != nil {
		log.Fatalf("Error creating adapter: %v", err)
	}
	defer adapter.Close()
	
	session, err := adapter.StartSession(0x400000)
	if err != nil {
		log.Fatalf("Error starting session: %v", err)
	}
	defer session.Close()
	
	packet := []byte("Hello, Wintun!")
	err = session.SendPacket(packet)
	if err != nil {
		log.Fatalf("Error sending packet: %v", err)
	}
	
	receivedPacket, err := session.ReceivePacket()
	if err != nil {
		log.Fatalf("Error receiving packet: %v", err)
	}
	fmt.Printf("Received packet: %s\n", receivedPacket)
}
```

## 📜 Documentation

For more detailed documentation on functions and methods available in the wintungo package, please refer to
the [GoDoc](https://pkg.go.dev/github.com/XenonCommunity/wintungo).

## 💡 Contributing

Contributions are welcome! If you have suggestions for improvements or features, please open an issue or submit a pull
request.

### Steps to Contribute:

1. Fork the repository.
2. Create a new branch: `git checkout -b my-feature`.
3. Make your changes and commit them: `git commit -m 'Add some feature'`.
4. Push to the branch: `git push origin my-feature`.
5. Open a pull request.

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🤝 Acknowledgments

- [wintun](https://www.wintun.net/) the wintun project.
- All contributors and community members for their support and contributions!

---

Happy coding! 🎊 Xenon Community 🎉
