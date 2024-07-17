package proof

import (
	"fmt"

	"github.com/cometbft/cometbft/crypto"

	"github.com/pokt-network/smt"
)

// GetNumComputeUnits returns the number of compute units for a given claim
// as determined by the sum of the root hash.
func (claim *Claim) GetNumComputeUnits() (numComputeUnits uint64, err error) {
	// NB: smt.MerkleRoot#Sum() will panic if the root hash is not valid.
	// Convert this panic into an error return.
	defer func() {
		if r := recover(); r != nil {
			numComputeUnits = 0
			err = fmt.Errorf(
				"unable to get sum of invalid merkle root: %x; error: %v",
				claim.GetRootHash(), r,
			)
		}
	}()

	return smt.MerkleRoot(claim.GetRootHash()).Sum(), nil
}

// GetNumRelays returns the number of relays for a given claim
// as determined by the count of the root hash.
func (claim *Claim) GetNumRelays() (numRelays uint64, err error) {
	// Convert this panic into an error return.
	defer func() {
		if r := recover(); r != nil {
			numRelays = 0
			err = fmt.Errorf(
				"unable to get count of invalid merkle root: %x; error: %v",
				claim.GetRootHash(), r,
			)
		}
	}()

	return smt.MerkleRoot(claim.GetRootHash()).Count(), nil
}

// GetHash returns the SHA-256 hash of the serialized claim.
func (claim *Claim) GetHash() ([]byte, error) {
	claimBz, err := claim.Marshal()
	if err != nil {
		return nil, err
	}

	return crypto.Sha256(claimBz), nil
}
