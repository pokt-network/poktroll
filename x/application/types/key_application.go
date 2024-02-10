package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix is the prefix to retrieve all Application
	ApplicationKeyPrefix = "Application/value/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(
	address string,
) []byte {
	var key []byte

	addressBytes := []byte(address)
	key = append(key, addressBytes...)
	key = append(key, []byte("/")...)

	return key
}
