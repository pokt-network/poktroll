package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SupplierKeyPrefix is the prefix to retrieve all Supplier
	SupplierKeyPrefix = "Supplier/address/"
)

// SupplierKey returns the store key to retrieve a Supplier from the index fields
func SupplierKey(supplierAddr string) []byte {
	var key []byte

	supplierAddrBz := []byte(supplierAddr)
	key = append(key, supplierAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
