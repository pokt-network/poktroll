package integration_test

import (
	"math"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	initialNumRelays = uint64(1e3)
	// DEV_NOTE: Max numRelays is set so that the test doesn't timeout.
	maxNumRelays = uint64(1024e3)
)

func TestComputeNewDifficultyHash_RewardsReflectWorkCompleted(t *testing.T) {
	// Update the target number of relays to a value that suits the test.
	// A too high number would make the difficulty stay at BaseRelayDifficultyHash
	initialTargetRelays := servicekeeper.TargetNumRelays
	servicekeeper.TargetNumRelays = 1000
	t.Cleanup(func() {
		// Reset the target number of relays to its initial value.
		servicekeeper.TargetNumRelays = initialTargetRelays
	})

	// Prepare the keepers and integration app
	integrationApp := integration.NewCompleteIntegrationApp(t)
	sdkCtx := integrationApp.GetSdkCtx()
	keepers := integrationApp.GetKeepers()

	app := integrationApp.DefaultApplication
	supplier := integrationApp.DefaultSupplier
	service := integrationApp.DefaultService

	// Set the CUTTM to 1 to simplify the math
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.ComputeUnitsToTokensMultiplier = uint64(1)
	err := keepers.SharedKeeper.SetParams(sdkCtx, sharedParams)
	require.NoError(t, err)

	// Set the global proof params so we never need a proof (for simplicity of this test)
	err = keepers.ProofKeeper.SetParams(sdkCtx, prooftypes.Params{
		ProofRequestProbability:   0,                                                                            // we never need a proof randomly
		ProofRequirementThreshold: &sdk.Coin{Denom: volatile.DenomuPOKT, Amount: sdkmath.NewInt(math.MaxInt64)}, // a VERY high threshold
	})
	require.NoError(t, err)

	// Update the relay mining difficulty so there's always a difficulty to retrieve
	// for the test service.
	_, err = keepers.ServiceKeeper.UpdateRelayMiningDifficulty(sdkCtx, map[string]uint64{service.Id: 0})
	require.NoError(t, err)

	// Set the previous relays and rewards to be used to calculate the increase ratio.
	previousNumRelays := uint64(0)
	previousRewards := sdkmath.NewInt(0)

	// Set the initial difficulty multiplier to later check that it has increased.
	difficultyMultiplier := big.NewRat(1, 1)

	// Monotonically increase the number of relays from a very small number
	// to a very large number.
	for numRelays := initialNumRelays; numRelays <= maxNumRelays; numRelays *= 2 {
		getSessionReq := sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: app.Address,
			ServiceId:          service.Id,
			BlockHeight:        sdkCtx.BlockHeight(),
		}
		sessionRes, err := keepers.SessionKeeper.GetSession(sdkCtx, &getSessionReq)
		require.NoError(t, err)

		session := sessionRes.Session

		// Determine the height at which the claim will expire.
		sessionEndToProofWindowCloseBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)
		sessionEndHeight := session.GetHeader().GetSessionEndBlockHeight()
		claimWindowOpenHeight := sessionEndHeight + int64(sessionEndToProofWindowCloseBlocks) + 1

		ctxAtHeight := sdkCtx.WithBlockHeight(claimWindowOpenHeight)

		// Get the relay mining difficulty that will be used when settling the pending claims.
		relayMiningDifficulty, ok := keepers.ServiceKeeper.GetRelayMiningDifficulty(ctxAtHeight, service.Id)
		require.True(t, ok)

		// Prepare a claim with the given number of relays.
		claim := prepareRealClaim(t, numRelays, supplier, session, service, &relayMiningDifficulty)

		// Get the claim's expected reward.
		claimedRewards, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
		require.NoError(t, err)

		// Get the number of claimed mined relays.
		claimNumRelays, err := claim.GetNumRelays()
		require.NoError(t, err)

		// Store the claim before settling it.
		keepers.ProofKeeper.UpsertClaim(ctxAtHeight, *claim)

		// Calling SettlePendingClaims calls ProcessTokenLogicModules behind the scenes
		settledResult, expiredResult, err := keepers.TokenomicsKeeper.SettlePendingClaims(ctxAtHeight)
		require.NoError(t, err)
		require.Equal(t, 1, int(settledResult.NumClaims))
		require.Equal(t, 0, int(expiredResult.NumClaims))

		// Update the relay mining difficulty
		_, err = keepers.TokenomicsKeeper.UpdateRelayMiningDifficulty(ctxAtHeight, map[string]uint64{service.Id: claimNumRelays})
		require.NoError(t, err)

		// Get the updated relay mining difficulty
		updatedRelayMiningDifficulty, ok := keepers.ServiceKeeper.GetRelayMiningDifficulty(ctxAtHeight, service.Id)
		require.True(t, ok)

		// Compute the new difficulty hash based on the updated relay mining difficulty.
		newDifficultyHash := protocol.ComputeNewDifficultyTargetHash(
			protocol.BaseRelayDifficultyHashBz,
			servicekeeper.TargetNumRelays,
			updatedRelayMiningDifficulty.NumRelaysEma,
		)

		// Check that the updated difficulty hash is correct.
		require.Equal(t, newDifficultyHash, updatedRelayMiningDifficulty.TargetHash)

		// Check that the new relays EMA has increased.
		require.Greater(t,
			updatedRelayMiningDifficulty.NumRelaysEma,
			relayMiningDifficulty.NumRelaysEma,
		)

		prevDifficultyMultiplier := protocol.GetRelayDifficultyMultiplier(relayMiningDifficulty.TargetHash)
		newDifficultyMultiplier := protocol.GetRelayDifficultyMultiplier(updatedRelayMiningDifficulty.TargetHash)
		// Check that the new difficulty has increased when the target number of relays is reached.
		if newDifficultyMultiplier.Cmp(big.NewRat(1, 1)) == 1 {
			require.True(t, newDifficultyMultiplier.Cmp(prevDifficultyMultiplier) == 0)
		}

		// Make sure that the rewards reflect the work completed and that it increases
		// proportionally to the number of relays mined.
		if previousNumRelays > 0 {
			numRelaysRatio := float64(numRelays) / float64(previousNumRelays)
			rewardsRatio, _ := new(big.Rat).SetFrac(claimedRewards.Amount.BigInt(), previousRewards.BigInt()).Float64()
			require.InDelta(t, numRelaysRatio, rewardsRatio, 0.05)
		}

		previousNumRelays = numRelays
		previousRewards = claimedRewards.Amount
		difficultyMultiplier = newDifficultyMultiplier
	}

	require.Equal(t, difficultyMultiplier.Cmp(big.NewRat(1, 1)), 1)
}

// prepareRealClaim prepares a claim by creating a real SMST with the given number
// of mined relays that adhere to the actual on-chain difficulty of the test service.
func prepareRealClaim(
	t *testing.T,
	numRelays uint64,
	supplier *sharedtypes.Supplier,
	session *sessiontypes.Session,
	service *sharedtypes.Service,
	relayMiningDifficulty *servicetypes.RelayMiningDifficulty,
) *prooftypes.Claim {
	t.Helper()

	// Prepare an in-memory key-value store
	kvStore, err := pebble.NewKVStore("")
	require.NoError(t, err)

	// Prepare an SMST
	trie := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

	// Insert the mined relays into the SMST
	for i := uint64(0); i < numRelays; i++ {
		// DEV_NOTE: Unsigned relays are mined instead of signed relays to avoid calling
		// the application querier and signature logic which make the test very slow
		// given the large number of iterations involved.
		minedRelay := testrelayer.NewUnsignedMinedRelay(t, session, supplier.OperatorAddress)
		// Ensure that the relay is applicable to the relay mining difficulty
		if protocol.IsRelayVolumeApplicable(minedRelay.Hash, relayMiningDifficulty.TargetHash) {
			err = trie.Update(minedRelay.Hash, minedRelay.Bytes, service.ComputeUnitsPerRelay)
			require.NoError(t, err)
		}
	}
	// Return the applicable claim
	return &prooftypes.Claim{
		SupplierOperatorAddress: supplier.OperatorAddress,
		SessionHeader:           session.GetHeader(),
		RootHash:                trie.Root(),
	}
}
