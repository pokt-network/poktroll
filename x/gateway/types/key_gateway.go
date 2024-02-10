package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// GatewayKeyPrefix is the prefix to retrieve all Gateways
	GatewayKeyPrefix = "Gateway/value/"
)

// GatewayKey returns the store key to retrieve a Gateway from the index fields
func GatewayKey(
	address string,
) []byte {
	var key []byte

	addressBytes := []byte(address)
	key = append(key, addressBytes...)
	key = append(key, []byte("/")...)

	return key
}
