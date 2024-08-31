package types

import (
	"github.com/cometbft/cometbft/crypto"
	"github.com/pokt-network/smt"
)

// GetNumComputeUnits returns the number of compute units for a given claim
// as determined by the sum of the root hash.
// TODO_MAINNET: Consider marking this function as deprecated to avoid confusion
// as to whether we should use "ComputeUnits" or "Relays*Service.ComputeUnitsPerRelay".
func (claim *Claim) GetNumComputeUnits() (numComputeUnits uint64, err error) {
	return smt.MerkleSumRoot(claim.GetRootHash()).Sum()
}

// GetNumRelays returns the number of relays for a given claim
// as determined by the count of the root hash.
func (claim *Claim) GetNumRelays() (numRelays uint64, err error) {
	return smt.MerkleSumRoot(claim.GetRootHash()).Count()
}

// GetHash returns the SHA-256 hash of the serialized claim.
func (claim *Claim) GetHash() ([]byte, error) {
	claimBz, err := claim.Marshal()
	if err != nil {
		return nil, err
	}

	return crypto.Sha256(claimBz), nil
}
