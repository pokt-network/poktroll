package types

import (
	"encoding/binary"
)

var _ binary.ByteOrder

const (
	// ClaimPrimaryKeyPrefix is the prefix to retrieve all Claim (the primary store)
	ClaimPrimaryKeyPrefix = "Claim/value/"

	// ClaimHeightPrefix is the key to retrieve a Claim's Primary Key from the Height index
	ClaimHeightPrefix = "Claim/height/"

	// ClaimAddressPrefix is the key to retrieve a Claim's Primary Key from the Address index
	ClaimAddressPrefix = "Claim/address/"

	// ClaimSessionIdPrefix is the key to retrieve a Claim's Primary Key from the SessionId index
	ClaimSessionIdPrefix = "Claim/sessionId/"
)

// ClaimPrimaryKey returns the primary store key to retrieve a Claim
func ClaimPrimaryKey(claim Claim) []byte {
	var key []byte

	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierAddr)
	key = append(key, []byte(claim.SessionId)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(claim.SupplierAddress)...)
	key = append(key, []byte("/")...)

	return key
}

// ClaimSupplierAddressKey returns the address key to iterate through claims given a supplier Address
func ClaimSupplierAddressKey(supplierAddr string) []byte {
	var key []byte

	key = append(key, []byte(supplierAddr)...)
	key = append(key, []byte("/")...)

	return key
}
