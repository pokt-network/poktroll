package types

import (
	"encoding/binary"
)

var _ binary.ByteOrder

const (

	// ClaimPrimaryKeyPrefix is the prefix to retrieve the entire Claim object (the primary store)
	ClaimPrimaryKeyPrefix = "Claim/value/"

	// ClaimSupplierAddressPrefix is the key to retrieve a Claim's Primary Key from the Address index
	ClaimSupplierAddressPrefix = "Claim/address/"

	// ClaimSessionEndHeightPrefix is the key to retrieve a Claim's Primary Key from the Height index
	ClaimSessionEndHeightPrefix = "Claim/height/"
)

// ClaimPrimaryKey returns the primary store key to retrieve a Claim by creating a composite key of the sessionId and supplierAddr
func ClaimPrimaryKey(sessionId, supplierAddr string) []byte {
	var key []byte

	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierAddr)
	// because every supplier can only have one claim per session.
	key = append(key, []byte(sessionId)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(supplierAddr)...)
	key = append(key, []byte("/")...)

	return key
}

// ClaimSupplierAddressKey returns the address key to iterate through claims given a supplier Address
func ClaimSupplierAddressKey(supplierAddr string, primaryKey []byte) []byte {
	var key []byte

	key = append(key, []byte(supplierAddr)...)
	key = append(key, []byte("/")...)
	key = append(key, primaryKey...)
	key = append(key, []byte("/")...)

	return key
}

// ClaimSupplierAddressKey returns the address key to iterate through claims given a supplier Address
func ClaimSupplierEndSessionHeightKey(sessionEndHeight uint64, primaryKey []byte) []byte {
	var key []byte

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, sessionEndHeight)

	key = append(key, []byte(heightBz)...)
	key = append(key, []byte("/")...)
	key = append(key, primaryKey...)
	key = append(key, []byte("/")...)

	return key
}
