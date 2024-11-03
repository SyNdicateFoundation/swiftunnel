package gateway

import "errors"

var (
	ErrNoGateway = errors.New("no gateway found")
	ErrCantParse = errors.New("unable to parse route output")
)
