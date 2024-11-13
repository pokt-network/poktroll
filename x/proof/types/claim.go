package types

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetNumClaimedComputeUnits returns the number of compute units for a given claim
// as determined by the sum of the root hash.
func (claim *Claim) GetNumClaimedComputeUnits() (numClaimedComputeUnits uint64, err error) {
	return smt.MerkleSumRoot(claim.GetRootHash()).Sum()
}

// GetNumRelays returns the number of relays for a given claim
// as determined by the count of the root hash.
func (claim *Claim) GetNumRelays() (numRelays uint64, err error) {
	return smt.MerkleSumRoot(claim.GetRootHash()).Count()
}

// GetNumEstimatedComputeUnits returns the claim's estimated number of compute units.
func (claim *Claim) GetNumEstimatedComputeUnits(
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) (numEstimatedComputeUnits uint64, err error) {
	numEstimatedComputeUnitsRat, err := claim.getNumEstimatedComputeUnitsRat(relayMiningDifficulty)
	if err != nil {
		return 0, err
	}

	numerator := numEstimatedComputeUnitsRat.Num()
	denominator := numEstimatedComputeUnitsRat.Denom()

	return new(big.Int).Div(numerator, denominator).Uint64(), nil
}

// GetClaimeduPOKT returns the claim's reward based on the relay mining difficulty
// and global network parameters.
func (claim *Claim) GetClaimeduPOKT(
	sharedParams sharedtypes.Params,
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) (sdk.Coin, error) {
	// Get the estimated number of compute units as a ratio to calculate the reward
	// to avoid precision loss.
	numEstimatedComputeUnitsRat, err := claim.getNumEstimatedComputeUnitsRat(relayMiningDifficulty)
	if err != nil {
		return sdk.Coin{}, err
	}

	computeUnitsToTokenMultiplierRat := new(big.Rat).SetUint64(sharedParams.GetComputeUnitsToTokensMultiplier())

	// CUTTM is a GLOBAL network wide parameter.
	upoktAmountRat := new(big.Rat).Mul(numEstimatedComputeUnitsRat, computeUnitsToTokenMultiplierRat)

	// Perform the division as late as possible to minimize precision loss.
	upoktAmount := new(big.Int).Div(upoktAmountRat.Num(), upoktAmountRat.Denom())
	if upoktAmount.Sign() < 0 {
		return sdk.Coin{}, ErrProofInvalidClaimedAmount.Wrapf(
			"num estimated compute units (%s) * CUTTM (%d) resulted in a negative amount: %s",
			numEstimatedComputeUnitsRat.RatString(),
			sharedParams.GetComputeUnitsToTokensMultiplier(),
			upoktAmountRat,
		)
	}

	return sdk.NewCoin(volatile.DenomuPOKT, math.NewIntFromBigInt(upoktAmount)), nil
}

// getNumEstimatedComputeUnitsRat returns the estimated claim's number of compute units
// as a ratio.
func (claim *Claim) getNumEstimatedComputeUnitsRat(
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) (numEstimatedComputeUnits *big.Rat, err error) {
	// Ensure the claim's service ID matches the relay mining difficulty service ID.
	if claim.GetSessionHeader().GetServiceId() != relayMiningDifficulty.GetServiceId() {
		return nil, ErrProofInvalidRelayDifficulty.Wrapf(
			"claim service ID (%s) does not match the service relay mining difficulty service ID (%s)",
			claim.GetSessionHeader().GetServiceId(),
			relayMiningDifficulty.GetServiceId(),
		)
	}

	numComputeUnits, err := claim.GetNumClaimedComputeUnits()
	if err != nil {
		return nil, err
	}

	numComputeUnitsRat := new(big.Rat).SetUint64(numComputeUnits)
	difficultyMultiplier := protocol.GetRelayDifficultyMultiplier(relayMiningDifficulty.GetTargetHash())
	numEstimatedComputeUnitsRat := new(big.Rat).Mul(difficultyMultiplier, numComputeUnitsRat)

	return numEstimatedComputeUnitsRat, nil
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
) (float64, error) {
	// Get the hash of the claim to seed the random number generator.
	claimHash, err := claim.GetHash()
	if err != nil {
		return 0, err
	}

	// Append the hash of the proofRequirementSeedBlockHash to the claim hash to seed
	// the random number generator to ensure that the proof requirement probability
	// is unknown until the proofRequirementSeedBlockHash is observed.
	proofRequirementSeed := append(claimHash, proofRequirementSeedBlockHash...)

	// Sample a pseudo-random value between [0,1) to determine if a proof is
	// required probabilistically.
	return poktrand.SeededFloat64(proofRequirementSeed), nil
}
