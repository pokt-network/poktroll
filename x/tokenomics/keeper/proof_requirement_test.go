package keeper_test

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

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
	// TODO_UPNEXT(#618): reuse requiredSampleSize()
	t.SkipNow()

	// Set expectedCompute units to be below the proof requirement threshold to only
	// exercise the probabilistic branch of the #isProofRequired() logic.
	expectedComputeUnits := prooftypes.DefaultProofRequirementThreshold - 1
	keepers, ctx := keeper.NewTokenomicsModuleKeepers(t, log.NewNopLogger())
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	var (
		sampleSize  = 15000
		probability = prooftypes.DefaultProofRequestProbability
		tolerance   = 0.01

		numTrueSamples atomic.Int64
	)

	// Sample concurrently to save time.
	wg := sync.WaitGroup{}
	for i := 0; i < sampleSize; i++ {
		wg.Add(1)
		go func() {
			claim := tetsproof.ClaimWithRandomHash(t, sample.AccAddress(), sample.AccAddress(), expectedComputeUnits)

			isRequired, err := keepers.Keeper.IsProofRequiredForClaim(sdkCtx, &claim)
			require.NoError(t, err)

			if isRequired {
				numTrueSamples.Add(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	expectedNumTrueSamples := float32(sampleSize) * probability
	expectedNumFalseSamples := float32(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)

	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := int64(sampleSize) - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples")
}
