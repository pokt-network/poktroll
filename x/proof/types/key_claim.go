package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ClaimPrimaryKeyPrefix is the prefix to retrieve the entire Claim object (the primary store)
	// TODO_TECHDEBT: consider renaming to ClaimSessionIDPrefix.
	ClaimPrimaryKeyPrefix = "Claim/primary_key/"

	// ClaimSupplierOperatorAddressPrefix is the key to retrieve a Claim's Primary Key from the Address index
	ClaimSupplierOperatorAddressPrefix = "Claim/address/"

	// ClaimSessionEndHeightPrefix is the key to retrieve a Claim's Primary Key from the Height index
	ClaimSessionEndHeightPrefix = "Claim/height/"
)

// ClaimPrimaryKey returns the primary store key used to retrieve a Claim by creating
// a composite key of the sessionId and supplierOperatorAddr.
func ClaimPrimaryKey(sessionId, supplierOperatorAddr string) []byte {
	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierOperatorAddr)
	// because every supplier can only have one claim per session.
	return KeyComposite([]byte(sessionId), []byte(supplierOperatorAddr))
}

// ClaimSupplierOperatorAddressKey returns the key used to iterate through claims given a supplier operator address.
func ClaimSupplierOperatorAddressKey(supplierOperatorAddr string, primaryKey []byte) []byte {
	return KeyComposite([]byte(supplierOperatorAddr), primaryKey)
}

// ClaimSupplierEndSessionHeightKey returns the key used to iterate through claims given a session end height.
func ClaimSupplierEndSessionHeightKey(sessionEndHeight int64, primaryKey []byte) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(sessionEndHeight))

	return KeyComposite(heightBz, primaryKey)
}

// TODO_TECHDEBT(@Olshansk): add helpers for composing query-side key prefixes & document key/value prefix design.
