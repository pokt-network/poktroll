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

// MicroToPicoPOKT is the conversion factor from uPOKT (micro POKT) to pPOKT (pico POKT).
// It is used to convert the estimated claim reward from pPOKT to uPOKT.
// See: https://en.wikipedia.org/wiki/Metric_prefix.
const MicroToPicoPOKT = 1e6

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

	// CUTTM is a GLOBAL network wide parameter.
	// computeUnitsToPpoktMultiplier allows for more granular compute unit costs, down to the pPOKT level.
	// It is used instead of a uPOKT multiplier to account for high POKT token prices
	// which might prevent low-cost services to be set at the appropriate (sub uPOKT) price point.
	computeUnitsToPpoktMultiplierRat := new(big.Rat).SetUint64(sharedParams.GetComputeUnitsToPpoktMultiplier())

	pPoktAmountRat := new(big.Rat).Mul(numEstimatedComputeUnitsRat, computeUnitsToPpoktMultiplierRat)

	// Perform the division as late as possible to minimize precision loss.
	ppoktAmount := new(big.Int).Div(pPoktAmountRat.Num(), pPoktAmountRat.Denom())

	// Convert the claim's reward from pPOKT to uPOKT.
	// This is done to ensure that the reward is always in uPOKT units,
	// DEV_NOTE: Although compute unit costs can be as small as 1 pPOKT, technically
	// a claim's reward MUST be at least 1 uPOKT.
	// Claims and Proofs transactions and submission fees would make such low reward
	// claims unprofitable anyway, and RelayMiners would perform profitability
	// checks before submitting such claims.
	// This approach also allows us to maintain consistency by using uPOKT throughout
	// the codebase rather than introducing pPOKT as a unit in other components.
	uPoktAmount := new(big.Int).Div(ppoktAmount, big.NewInt(MicroToPicoPOKT))

	if uPoktAmount.Sign() < 0 {
		return sdk.Coin{}, ErrProofInvalidClaimedAmount.Wrapf(
			"num estimated compute units (%s) * CUTTM (%d) resulted in a negative amount: %s",
			numEstimatedComputeUnitsRat.RatString(),
			sharedParams.GetComputeUnitsToPpoktMultiplier(),
			uPoktAmount,
		)
	}

	return sdk.NewCoin(volatile.DenomuPOKT, math.NewIntFromBigInt(uPoktAmount)), nil
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
