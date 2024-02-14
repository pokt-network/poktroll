package types

import (
	"encoding/binary"
	fmt "fmt"
)

var _ binary.ByteOrder

const (
	// ServiceKeyPrefix is the prefix to retrieve all Service
	SessionKeyPrefix = "Session/value/"
)

// ServiceKey returns the store key to retrieve a Service from the index fields
func SessionKey(blockHeight int64) []byte {
	var key []byte

	serviceIDBz := []byte(fmt.Sprintf("%d", blockHeight))
	key = append(key, serviceIDBz...)
	key = append(key, []byte("/")...)

	return key
}
