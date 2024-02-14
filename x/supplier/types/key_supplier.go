package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SupplierKeyPrefix is the prefix to retrieve all Supplier
	SupplierKeyPrefix = "Supplier/value/"
)

// SupplierKey returns the store key to retrieve a Supplier from the index fields
func SupplierKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
