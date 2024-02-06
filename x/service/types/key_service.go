package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ServiceKeyPrefix is the prefix to retrieve all Service
	ServiceKeyPrefix = "Service/value/"
)

// ServiceKey returns the store key to retrieve a Service from the index fields
func ServiceKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
