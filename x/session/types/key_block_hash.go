package types

import (
	"encoding/binary"
	"fmt"
)

var _ binary.ByteOrder

const (
	// BlockHashKeyPrefix is the prefix to retrieve all BlockHash
	BlockHashKeyPrefix = "BlockHash/value/"
)

// BlockHashKey returns the store key to retrieve a BlockHash from the index fields
func BlockHashKey(blockHeight int64) []byte {
	var key []byte

	serviceIDBz := []byte(fmt.Sprintf("%d", blockHeight))
	key = append(key, serviceIDBz...)
	key = append(key, []byte("/")...)

	return key
}
