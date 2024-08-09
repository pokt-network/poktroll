package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SupplierKeyOperatorPrefix is the prefix to retrieve all Supplier
	SupplierKeyOperatorPrefix = "Supplier/operator_address/"
)

// SupplierOperatorKey returns the store key to retrieve a Supplier from the index fields
func SupplierOperatorKey(supplierOperatorAddr string) []byte {
	var key []byte

	supplierOperatorAddrBz := []byte(supplierOperatorAddr)
	key = append(key, supplierOperatorAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
