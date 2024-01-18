package types

import (
	"encoding/binary"
)

var _ binary.ByteOrder

const (
	// RedelegationPrimaryKeyPrefix is the prefix to retrieve all Redelegations.
	RedelegationPrimaryKeyPrefix = "Redelgation/value/"

	// RedelegationCompletionPrimaryKeyPrefix is the prefix to retrieve all
	// Redelegations ordered by completion block height.
	RedelegationCompletionPrimaryKeyPrefix = "Redelgation/completion/"
)

// RedelegationPrimaryKey returns the primary store key used to retrieve a
// Redelegation by creating a composite key of the appAddr and gatewayAddr
// and the blockHeight at initiation and the completion blockHeight.
func RedelegationPrimaryKey(
	appAddr, gatewayAddr string,
	redlegationID uint64,
) []byte {
	redlegationIDBz := make([]byte, 8)
	binary.BigEndian.PutUint64(redlegationIDBz, redlegationID)
	return KeyComposite(
		[]byte(appAddr),
		[]byte(gatewayAddr),
		redlegationIDBz,
	)
}

// RedelegationCompletionPrimaryKey returns the primary store key used to
// retrieve a Redelegation by the completionBlockHeight, appAddr and gatewayAddr
// and the blockHeight at initiation.
func RedelegationCompletionPrimaryKey(
	appAddr, gatewayAddr string,
	blockHeight, completionBlockHeight int64,
	redlegationID uint64,
) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(blockHeight))
	completionHeightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(completionHeightBz, uint64(completionBlockHeight))
	redelegationIDBz := make([]byte, 8)
	binary.BigEndian.PutUint64(redelegationIDBz, uint64(0))
	return KeyComposite(
		completionHeightBz,
		heightBz,
		redelegationIDBz,
		[]byte(appAddr),
		[]byte(gatewayAddr),
	)
}
