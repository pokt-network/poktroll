package types

import (
	"encoding/binary"
)

var _ binary.ByteOrder

const (
	// ServiceKeyPrefix is the prefix to retrieve all Services
	ServiceKeyPrefix = "Service/value/"
)

// ServiceKey returns the store key to retrieve a Service from the index fields
func ServiceKey(serviceID string) []byte {
	var key []byte

	serviceIDBytes := []byte(serviceID)
	key = append(key, serviceIDBytes...)
	key = append(key, []byte("/")...)

	return key
}
