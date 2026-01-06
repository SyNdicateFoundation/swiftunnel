package gateway

import "errors"

// Common errors for gateway discovery.
var (
	ErrNoGateway = errors.New("no gateway found")
	ErrCantParse = errors.New("unable to parse route output")
)
