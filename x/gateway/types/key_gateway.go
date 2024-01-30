package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// GatewayKeyPrefix is the prefix to retrieve all Gateway
	GatewayKeyPrefix = "Gateway/value/"
)

// GatewayKey returns the store key to retrieve a Gateway from the index fields
func GatewayKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
