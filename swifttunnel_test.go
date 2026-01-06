//go:build windows

package swiftunnel

import (
	"github.com/SyNdicateFoundation/swiftunnel/swiftconfig"
	"github.com/SyNdicateFoundation/swiftunnel/swiftypes"
	"log"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	// Create a default configuration
	config, err := swiftconfig.New(
		swiftconfig.WithUnicastIP("10.18.21.1/24"),
	)
	if err != nil {
		t.Fatal(err)
	}

	config.AdapterGUID, err = swiftypes.ParseGUID("ab9e3a03-de9f-4ce9-89e5-b962aab6d3f0")

	s, err := NewSwiftInterface(config)
	if err != nil {
		t.Fatal(err)
	}

	defer s.Close()

	if err := s.SetStatus(swiftypes.InterfaceDown); err != nil {
		t.Fatal(err)
	}

	log.Println("InterfaceDown")

	if err := s.SetStatus(swiftypes.InterfaceUp); err != nil {
		t.Fatal(err)
	}

	log.Println("InterfaceUp")
}
