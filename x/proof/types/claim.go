package types

import (
	"github.com/cometbft/cometbft/crypto"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	"github.com/pokt-network/smt"
)

// GetNumComputeUnits returns the number of compute units for a given claim
// as determined by the sum of the root hash.
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

// GetProofRequirementSampleValue returns a pseudo-random value between 0 and 1 to
// determine if a proof is required probabilistically.
// IMPORTANT: It is assumed that the caller has ensured the hash of the block seed
func (claim *Claim) GetProofRequirementSampleValue(
	proofRequirementSeedBlockHash []byte,
) (proofRequirementSampleValue float32, err error) {
	// Get the hash of the claim to seed the random number generator.
	var claimHash []byte
	claimHash, err = claim.GetHash()
	if err != nil {
		return 0, err
	}

	// Append the hash of the proofRequirementSeedBlockHash to the claim hash to seed
	// the random number generator to ensure that the proof requirement probability
	// is unknown until the proofRequirementSeedBlockHash is observed.
	proofRequirementSeed := append(claimHash, proofRequirementSeedBlockHash...)

	// Sample a pseudo-random value between 0 and 1 to determine if a proof is
	// required probabilistically.
	proofRequirementSampleValue, err = poktrand.SeededFloat32(proofRequirementSeed)
	if err != nil {
		return 0, err
	}

	return proofRequirementSampleValue, nil
}
