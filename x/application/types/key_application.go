package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix is the prefix to retrieve all Application
	ApplicationKeyPrefix = "Application/address/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(appAddr string) []byte {
	var key []byte

	appAddrBz := []byte(appAddr)
	key = append(key, appAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
