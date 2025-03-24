package keeper_test

import (
	"sync/atomic"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	"github.com/pokt-network/poktroll/testutil/keeper"
	tetsproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/types"
)

func TestKeeper_IsProofRequired(t *testing.T) {
	keepers, ctx := keeper.NewProofModuleKeepers(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	proofParams := keepers.Keeper.GetParams(sdkCtx)
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	// Set expected compute units to be below the proof requirement threshold to only
	// exercise the probabilistic branch of the #isProofRequired() logic.
	expectedComputeUnits := (proofParams.ProofRequirementThreshold.Amount.Uint64() - 1) / sharedParams.ComputeUnitsToTokensMultiplier

	var (
		probability = types.DefaultProofRequestProbability
		// This was empirically determined to avoid false negatives in unit tests.
		// As a maintainer of the codebase, you may need to adjust these.
		tolerance  = 0.10
		confidence = 0.98

		numTrueSamples atomic.Int64
	)

	// TODO_TECHDEBT(@bryanchriswhite): This test is periodically flaky but theoretically shouldn't be.
	// What can we do to increase it's consistency without diving tolerance by 2?
	sampleSize := poktrand.RequiredSampleSize(float64(probability), tolerance/2, confidence)

	// NB: Not possible to sample concurrently, this causes a race condition due to the keeper's gas meter.
	for i := int64(0); i < sampleSize; i++ {
		claim := tetsproof.ClaimWithRandomHash(t, sample.AccAddress(), sample.AccAddress(), expectedComputeUnits)

		proofRequirementReason, err := keepers.ProofRequirementForClaim(sdkCtx, &claim)
		require.NoError(t, err)

		if proofRequirementReason != types.ProofRequirementReason_NOT_REQUIRED {
			numTrueSamples.Add(1)
		}
	}

	expectedNumTrueSamples := float64(sampleSize) * probability
	expectedNumFalseSamples := float64(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)
	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples not in range")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples not in range")
}
