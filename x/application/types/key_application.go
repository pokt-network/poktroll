package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix is the prefix to retrieve all Application
	ApplicationKeyPrefix = "Application/address/"

	// ApplicationUnstakingKeyPrefix is the prefix for indexing unstaking applications
	ApplicationUnstakingKeyPrefix = "Application/unstaking/"

	// ApplicationTransferKeyPrefix is the prefix for indexing applications that
	// are being transferred
	ApplicationTransferKeyPrefix = "Application/transfer/"

	// ApplicationUnbondingKeyPrefix is the prefix for indexing pending undelegations
	UndelegationKeyPrefix = "Application/undelegation/"

	// ApplicationDelegationKeyPrefix is the prefix for indexing applications that are
	// delegating to gateways
	DelegationKeyPrefix = "Application/delegation/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(appAddr string) []byte {
	return StringKey(appAddr)
}

// UndelegationKey returns the store key to retrieve an undelegation from the index fields.
// The key is composed of the application address and the gateway address.
// This ordering allows efficient range queries for undlegations by application address.
func UndelegationKey(appAddr, gatewayAddr string) []byte {
	var key []byte

	appKey := StringKey(appAddr)
	key = append(key, appKey...)

	gatewayKey := StringKey(gatewayAddr)
	key = append(key, gatewayKey...)

	return key
}

// DelegationKey returns the store key to retrieve a delegation from the index fields.
// The key is composed of the gateway address and the application address.
// This ordering allows efficient range queries for delegations by gateway address.
func DelegationKey(gatewayAddr, appAddr string) []byte {
	var key []byte

	gatewayKey := StringKey(gatewayAddr)
	key = append(key, gatewayKey...)

	appAddrKey := StringKey(appAddr)
	key = append(key, appAddrKey...)

	return key
}

// IntKey converts an interger value to a byte slice for use in store keys
// Appends a "/" separator to the end of the key for consistent prefix scanning
func IntKey(intIndex int64) []byte {
	var key []byte

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(intIndex))
	key = append(key, heightBz...)
	key = append(key, []byte("/")...)

	return key
}

// StringKey converts a string value to a byte slice for use in store keys
// Appends a "/" separator to the end of the key for consistent prefix scanning
func StringKey(strIndex string) []byte {
	var key []byte

	strIndexBz := []byte(strIndex)
	key = append(key, strIndexBz...)
	key = append(key, []byte("/")...)

	return key
}
