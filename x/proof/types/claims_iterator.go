package types

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// ClaimsIterator provides iteration functionality over claims stored in the blockchain state.
// It wraps the underlying store iterator, primary store, and codec to provide convenient access to Claim objects.
// The iterator uses a two-level storage approach where the main iterator provides keys that reference the actual
// claim data in the primary store.
type ClaimsIterator struct {
	primaryStore storetypes.KVStore
	iterator     storetypes.Iterator
	cdc          codec.BinaryCodec
}

// Next advances the iterator to the next position.
func (ci *ClaimsIterator) Next() {
	ci.iterator.Next()
}

// Value returns the current Claim that the iterator is pointing to. It:
// 1. Retrieves the primary key from the iterator's current value
// 2. Uses the primary key above to fetch the actual claim data from the primary store
// 3. Unmarshals the retrieve bytes data into a Claim object
func (ci *ClaimsIterator) Value() *Claim {
	primaryKey := ci.iterator.Value()
	claimBz := ci.primaryStore.Get(primaryKey)
	var claim Claim
	ci.cdc.MustUnmarshal(claimBz, &claim)
	return &claim
}

// Close releases any resources held by the iterator.
// It should be called when the iterator is no longer needed.
func (ci *ClaimsIterator) Close() {
	ci.iterator.Close()
}

// Valid returns whether the iterator is still valid and can continue to be used.
// Returns false when the iterator has been exhausted or closed.
func (ci *ClaimsIterator) Valid() bool {
	return ci.iterator.Valid()
}

// NewClaimsIterator creates a new ClaimsIterator instance.
func NewClaimsIterator(
	iterator storetypes.Iterator,
	primaryStore storetypes.KVStore,
	cdc codec.BinaryCodec,
) *ClaimsIterator {
	return &ClaimsIterator{
		iterator:     iterator,
		primaryStore: primaryStore,
		cdc:          cdc,
	}
}
