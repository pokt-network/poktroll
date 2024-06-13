package keeper_test

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestKeeper_IsProofRequired(t *testing.T) {
	// Set expectedCompute units to be below the proof requirement threshold to only
	// exercise the probabilistic branch of the #isProofRequired() logic.
	expectedComputeUnits := prooftypes.DefaultProofRequirementThreshold - 1
	keepers, ctx := keeper.NewTokenomicsModuleKeepers(t, log.NewNopLogger())
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	var (
		// Because this test is deterministic & this sample size is known to be
		// sufficient, it doest not need to be calculated.
		sampleSize  = 1500
		samples     = make(map[bool]int64)
		probability = prooftypes.DefaultProofRequestProbability
		tolerance   = 0.01
	)

	for i := 0; i < sampleSize; i++ {
		claim := claimWithRandomHash(t, sample.AccAddress(), sample.AccAddress(), expectedComputeUnits)

		isRequired, err := keepers.Keeper.IsProofRequiredForClaim(sdkCtx, &claim)
		require.NoError(t, err)

		samples[isRequired]++
	}

	// Check that the number of samples for each outcome is within the expected range.
	for outcome, count := range samples {
		t.Run(fmt.Sprintf("outcome_%t", outcome), func(t *testing.T) {
			var expectedCount float32
			switch outcome {
			case true:
				expectedCount = float32(sampleSize) * probability
			case false:
				expectedCount = float32(sampleSize) * (1 - probability)
			}

			require.InDeltaf(t, expectedCount, count, tolerance*float64(sampleSize), "outcome: %t", outcome)
		})
	}
}

func claimWithRandomHash(t *testing.T, appAddr, supplierAddr string, sum uint64) prooftypes.Claim {
	claim := baseClaim(appAddr, supplierAddr, sum)
	claim.RootHash = randSmstRootWithSum(t, sum)
	return claim
}

func randSmstRootWithSum(t *testing.T, sum uint64) smt.MerkleRoot {
	t.Helper()

	root := make([]byte, 40)
	// Only populate the first 32 bytes with random data, leave the last 8 bytes for the sum.
	_, err := rand.Read(root[:32])
	require.NoError(t, err)

	binary.BigEndian.PutUint64(root[32:], sum)
	return smt.MerkleRoot(root)
}
