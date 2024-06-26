package keeper_test

import (
	"math/rand"
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

// NB: This init function is used to seed the random number generator to ensure
// that the test is deterministic.
func init() {
	rand.Seed(0)
}

func TestKeeper_IsProofRequired(t *testing.T) {
	// Set expectedCompute units to be below the proof requirement threshold to only
	// exercise the probabilistic branch of the #isProofRequired() logic.
	expectedComputeUnits := prooftypes.DefaultProofRequirementThreshold - 1
	keepers, ctx := keeper.NewTokenomicsModuleKeepers(t, log.NewNopLogger())
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	var (
		probability = prooftypes.DefaultProofRequestProbability
		tolerance   = 0.01
		confidence  = 0.99

		numTrueSamples atomic.Int64
	)

	sampleSize := poktrand.RequiredSampleSize(float64(probability), tolerance, confidence)

	// NB: Not possible to sample concurrently, this causes a race condition due to the keeper's gas meter.
	for i := int64(0); i < sampleSize; i++ {
		claim := tetsproof.ClaimWithRandomHash(t, sample.AccAddress(), sample.AccAddress(), expectedComputeUnits)

		isRequired, err := keepers.Keeper.IsProofRequiredForClaim(sdkCtx, &claim)
		require.NoError(t, err)

		if isRequired {
			numTrueSamples.Add(1)
		}
	}

	expectedNumTrueSamples := float32(sampleSize) * probability
	expectedNumFalseSamples := float32(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)

	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples")
}
