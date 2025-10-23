# Swifttunnel

## Getting Started 🚧

## Installation

To install Swiftunnel, follow these steps:

1. Ensure you have Go 1.16 or later installed on your system.
2. Clone the Swiftunnel repository:
   ```
   git clone https://github.com/SyNdicateFoundation/swiftunnel.git
   ```
3. Change into the project directory:
   ```
   cd swiftunnel
   ```
4. Build the Swiftunnel binary:
   ```
   go build -o swiftunnel ./cmd/swiftunnel
   ```

## Usage

To use Swiftunnel, you can create a new `Config` struct and pass it to the `NewSwiftInterface` function:

```go
config := swiftunnel.NewDefaultConfig()
adapter, err := swiftunnel.NewSwiftInterface(config)
if err != nil {
    // Handle error
}
defer adapter.Close()

// Use the adapter
```

The `Config` struct allows you to customize the adapter settings, such as the adapter name, MTU, and IP address configuration.

## Library

The Swiftunnel package provides the following main types and functions:

- `Config`: Represents the configuration for a Swiftunnel adapter.
- `NewSwiftInterface(config Config) (*SwiftInterface, error)`: Creates a new Swiftunnel adapter based on the provided configuration.
- `SwiftInterface`: Represents a Swiftunnel adapter, implementing the `io.ReadWriteCloser` interface.
- `SetMTU(mtu int) error`: Sets the MTU (Maximum Transmission Unit) of the adapter.
- `SetUnicastIpAddressEntry(config *UnicastConfig) error`: Sets the unicast IP address configuration for the adapter.


## Contribution Guidelines 🤝

Feel free to contribute to the development of our bot. we will notice it.
