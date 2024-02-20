package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix is the prefix to retrieve all Application
	ApplicationKeyPrefix = "Application/value/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(address string) []byte {
	var key []byte

	addressBz := []byte(address)
	key = append(key, addressBz...)
	key = append(key, []byte("/")...)

	return key
}
