package keeper_test

import (
	"sync/atomic"
	"testing"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	"github.com/pokt-network/poktroll/testutil/keeper"
	tetsproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestKeeper_IsProofRequired(t *testing.T) {
	keepers, ctx := keeper.NewTokenomicsModuleKeepers(t, log.NewNopLogger())
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	proofParams := keepers.ProofKeeper.GetParams(sdkCtx)
	tokenomicsParams := keepers.Keeper.GetParams(sdkCtx)
	// Set expectedCompute units to be below the proof requirement threshold to only
	// exercise the probabilistic branch of the #isProofRequired() logic.
	expectedComputeUnits := (proofParams.ProofRequirementThreshold.Amount.Uint64() - 1) / tokenomicsParams.ComputeUnitsToTokensMultiplier

	var (
		probability = prooftypes.DefaultProofRequestProbability
		// This was empirically determined to avoid false negatives in unit tests.
		// As a maintainer of the codebase, you may need to adjust these.
		tolerance  = 0.10
		confidence = 0.98

		numTrueSamples atomic.Int64
	)

	// TODO_BETA(@bryanchriswhite): This test is periodically flaky but theoretically shouldn't be.
	// What can we do to increase it's consistency without diving tolerance by 2?
	sampleSize := poktrand.RequiredSampleSize(float64(probability), tolerance/2, confidence)

	// NB: Not possible to sample concurrently, this causes a race condition due to the keeper's gas meter.
	for i := int64(0); i < sampleSize; i++ {
		claim := tetsproof.ClaimWithRandomHash(t, sample.AccAddress(), sample.AccAddress(), expectedComputeUnits)

		proofRequirementReason, err := keepers.Keeper.ProofRequirementForClaim(sdkCtx, &claim)
		require.NoError(t, err)

		if proofRequirementReason != prooftypes.ProofRequirementReason_NOT_REQUIRED {
			numTrueSamples.Add(1)
		}
	}

	expectedNumTrueSamples := float32(sampleSize) * probability
	expectedNumFalseSamples := float32(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)
	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples not in range")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples not in range")
}
