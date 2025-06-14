package types

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetNumClaimedComputeUnits returns the number of compute units for a given claim.
// It is determined by the sum stored in the root hash of the SMST.
func (claim *Claim) GetNumClaimedComputeUnits() (numClaimedComputeUnits uint64, err error) {
	return smt.MerkleSumRoot(claim.GetRootHash()).Sum()
}

// GetNumRelays returns the number of relays for a given claim.
// It is the count of non-empty leaves in the tree.
//
// Note that not every Relay (Request, Response) pair in the session is inserted into the tree.
// The relay hash has to have matched the difficulty for that service.
//
// This is controlled by the Relay Mining difficulty to reduce co-processor
// hardware requirements and enable scaling to tens of billions of relays.
func (claim *Claim) GetNumRelays() (numRelays uint64, err error) {
	return smt.MerkleSumRoot(claim.GetRootHash()).Count()
}

// GetNumEstimatedComputeUnits returns the claim's estimated number of compute units.
// numEstimatedComputeUnits =
func (claim *Claim) GetNumEstimatedComputeUnits(
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) (numEstimatedComputeUnits uint64, err error) {
	numEstimatedComputeUnitsRat, err := claim.getNumEstimatedComputeUnitsRat(relayMiningDifficulty)
	if err != nil {
		return 0, err
	}

	// Safe (high-precision) float division
	numerator := numEstimatedComputeUnitsRat.Num()
	denominator := numEstimatedComputeUnitsRat.Denom()
	return new(big.Int).Div(numerator, denominator).Uint64(), nil
}

// GetClaimeduPOKT returns the claim's token reward in uPOKT.
// At a high-level, the following is done:
// estimatedOffchainComputeUnits = claim.NumVolumeApplicableComputeUnits * service.RelayMiningDifficulty
// uPOKT = estimatedOffchainComputeUnits * chain.ComputeUnitsToTokenMultiplier / chain.ComputeUnitsCostGranularity
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

	// Calculate the fractional uPOKT amount of a single compute unit.
	computeUnitsToUpoktMultiplierRat := new(big.Rat).SetFrac64(
		// CUTTM is a GLOBAL network wide parameter.
		int64(sharedParams.GetComputeUnitsToTokensMultiplier()),

		// The uPOKT cost granularity of a single compute unit.
		int64(sharedParams.GetComputeUnitCostGranularity()),
	)

	// Perform the division as late as possible to minimize precision loss.
	upoktAmountRat := new(big.Rat).Mul(numEstimatedComputeUnitsRat, computeUnitsToUpoktMultiplierRat)
	upoktAmount := new(big.Int).Div(upoktAmountRat.Num(), upoktAmountRat.Denom())

	// Sanity check against unpredictable errors
	if upoktAmount.Sign() < 0 {
		return sdk.Coin{}, ErrProofInvalidClaimedAmount.Wrapf(
			"SHOULD NEVER HAPPEN: num estimated compute units (%s) * CUTTM (%s) resulted in a negative amount: %s",
			numEstimatedComputeUnitsRat.RatString(),
			computeUnitsToUpoktMultiplierRat.RatString(),
			upoktAmountRat,
		)
	}

	return sdk.NewCoin(pocket.DenomuPOKT, math.NewIntFromBigInt(upoktAmount)), nil
}

// getNumEstimatedComputeUnitsRat returns the claim's estimated number of compute units
// as a big.Rat ratio.
// This is necessary for safe (high-precision) float division.
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

	// Retrieve the number of compute unit's from the root hash corresponding to the claim's SMST.
	numComputeUnits, err := claim.GetNumClaimedComputeUnits()
	if err != nil {
		return nil, err
	}

	// This is necessary for safe (high-precision) float division.
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

// GetDehydratedClaim returns a copy of the claim with only the essential fields.
func (claim *Claim) GetDehydratedClaim() Claim {
	return Claim{
		SupplierOperatorAddress: claim.SupplierOperatorAddress,
		SessionHeader:           claim.SessionHeader,
	}
}
