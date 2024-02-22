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

// ClaimPrimaryKey returns the primary store key used to retrieve a Claim by creating a composite key of the sessionId and supplierAddr.
func ClaimPrimaryKey(sessionId, supplierAddr string) []byte {
	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierAddr)
	// because every supplier can only have one claim per session.
	return KeyComposite([]byte(sessionId), []byte(supplierAddr))
}

// ClaimSupplierAddressKey returns the key used to iterate through claims given a supplier Address.
func ClaimSupplierAddressKey(supplierAddr string, primaryKey []byte) []byte {
	return KeyComposite([]byte(supplierAddr), primaryKey)
	//claim/address/1 -> key1
	//claim/address/2 -> key2
	//claim/address/3 -> key3
	//claim/address/4 -> key4
	//claim/address/5 -> key5
	//claim/address/{addr}/* -> key*
	//claim/height/{height}* -> key*
}

// ClaimSupplierEndSessionHeightKey returns the key used to iterate through claims given a session end height.
func ClaimSupplierEndSessionHeightKey(sessionEndHeight int64, primaryKey []byte) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(sessionEndHeight))

	return KeyComposite(heightBz, primaryKey)
}
