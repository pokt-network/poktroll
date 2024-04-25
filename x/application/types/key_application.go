package types

import (
	"encoding/binary"
	fmt "fmt"
)

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix is the prefix to retrieve all Application
	ApplicationKeyPrefix = "Application/address/"
	// PendingUndelegationsKeyPrefix is the prefix to retrieve all PendingUndelegations
	PendingUndelegationsKeyPrefix = "PendingUndelegations/appAddress/gatewayAddress/"
	// ArchivedDelegationsKeyPrefix is the prefix to retrieve all existing archived delegations
	ArchivedDelegationsKeyPrefix = "ArchivedDelegations/blockHeight/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(appAddr string) []byte {
	var key []byte

	appAddrBz := []byte(appAddr)
	key = append(key, appAddrBz...)
	key = append(key, []byte("/")...)

	return key
}

// PendingUndelegationKey returns the store key to retrieve a PendingUndelegation.
// It is the concatenation of the AppAddress and GatewayAddress.
func PendingUndelegationKey(pendingUndelegation *Undelegation) []byte {
	var key []byte

	appAddrBz := []byte(pendingUndelegation.AppAddress)
	gatewayAddrBz := []byte(pendingUndelegation.GatewayAddress)

	key = append(key, appAddrBz...)
	key = append(key, []byte("/")...)

	key = append(key, gatewayAddrBz...)
	key = append(key, []byte("/")...)

	return key
}

// ArchivedDelegationsBlockKey returns the store key to retrieve the application
// addresses with archived delegations that happened at the given block height.
func ArchivedDelegationsBlockKey(blockHeight int64) []byte {
	var key []byte

	appAddrBz := []byte(fmt.Sprintf("%d", blockHeight))
	key = append(key, appAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
