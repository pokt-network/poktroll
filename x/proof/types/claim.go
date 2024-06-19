package types

import "github.com/pokt-network/smt"

// GetNumComputeUnits returns the number of compute units for a given claim
// as determined by the sum of the root hash.
func (claim *Claim) GetNumComputeUnits() uint64 {
	return smt.MerkleRoot(claim.GetRootHash()).Sum()
}

// GetNumRelays returns the number of relays for a given claim
// as determined by the count of the root hash.
func (claim *Claim) GetNumRelays() uint64 {
	return smt.MerkleRoot(claim.GetRootHash()).Count()
}
