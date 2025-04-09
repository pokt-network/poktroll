package types

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// ClaimsIterator provides iteration functionality over claims stored in the blockchain state.
// It wraps the underlying store iterator and codec to provide convenient access to Claim objects.
type ClaimsIterator struct {
	iterator storetypes.Iterator
	cdc      codec.BinaryCodec
}

// Next advances the iterator to the next position.
func (ci *ClaimsIterator) Next() {
	ci.iterator.Next()
}

// Value returns the current Claim that the iterator is pointing to.
// It unmarshals the raw bytes from the store into a Claim object.
func (ci *ClaimsIterator) Value() *Claim {
	var claim Claim
	ci.cdc.MustUnmarshal(ci.iterator.Value(), &claim)
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
func NewClaimsIterator(iterator storetypes.Iterator, cdc codec.BinaryCodec) *ClaimsIterator {
	return &ClaimsIterator{
		iterator: iterator,
		cdc:      cdc,
	}
}
