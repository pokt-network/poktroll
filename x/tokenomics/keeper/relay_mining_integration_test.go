package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestComputeNewDifficultyHash_RewardsReflectWorkCompleted(t *testing.T) {
	// Test params
	appInitialStake := math.NewInt(1000000)
	supplierInitialStake := math.NewInt(1000000)
	globalComputeUnitsToTokensMultiplier := uint64(1) // keeping the math simple
	serviceComputeUnitsPerRelay := uint64(1)          // keeping the math simple
	service := prepareTestService(serviceComputeUnitsPerRelay)

	// Prepare the keepers
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, testkeeper.WithService(*service))
	keepers.SetService(ctx, *service)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1)

	// Set the global tokenomics params
	err := keepers.Keeper.SetParams(sdkCtx, tokenomicstypes.Params{
		ComputeUnitsToTokensMultiplier: globalComputeUnitsToTokensMultiplier,
	})
	require.NoError(t, err)

	// Set the global proof params so we never need a proof (for simplicity of this test)
	err = keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,            // we never need a proof randomly
		ProofRequirementThreshold: uint64(1e18), // a VERY high threshold
	})
	require.NoError(t, err)

	// Add a new application with non-zero app stake
	appStake := cosmostypes.NewCoin(volatile.DenomuPOKT, appInitialStake)
	app := apptypes.Application{
		Address:        sample.AccAddress(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{Service: service}},
	}
	keepers.SetApplication(sdkCtx, app)

	// Add a new application with non-zero supplier stake
	supplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, supplierInitialStake)
	supplierAddr := sample.AccAddress()
	revShare := &sharedtypes.ServiceRevenueShare{
		Address:            supplierAddr,
		RevSharePercentage: 100,
	}
	supplier := sharedtypes.Supplier{
		OwnerAddress:    supplierAddr,
		OperatorAddress: supplierAddr,
		Stake:           &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{{
			Service:  service,
			RevShare: []*sharedtypes.ServiceRevenueShare{revShare},
		}},
	}
	keepers.SetSupplier(sdkCtx, supplier)

	// TODO_TECHDEBT: Determine the height at which the claim will expire.
	// Since "prepareTestClaim" starts every session at one, we can just create
	// claims and keep settling them at the same height.
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := shared.GetSessionEndHeight(&sharedParams, 1)
	claimWindowSizeBlocks := int64(sharedParams.GetClaimWindowOpenOffsetBlocks() + sharedParams.GetClaimWindowCloseOffsetBlocks())
	proofWindowSizeBlocks := int64(sharedParams.GetProofWindowOpenOffsetBlocks() + sharedParams.GetProofWindowCloseOffsetBlocks())
	claimExpirationHeight := sessionEndHeight + claimWindowSizeBlocks + proofWindowSizeBlocks + 1
	sdkCtx = sdkCtx.WithBlockHeight(claimExpirationHeight)

	// Num relays is monotonically increasing to a large number
	for numRelays := uint64(1e3); numRelays <= 1e16; numRelays *= 10 {
		// trie := prepareSMST(t, sdkCtx, integrationApp, session, expectedNumRelays)
		// Prepare a claim with the given number of relays and store it
		claim := prepareRealClaim(numRelays, service, &app, &supplier)
		keepers.ProofKeeper.UpsertClaim(sdkCtx, claim)

		// Calling SettlePendingClaims calls ProcessTokenLogicModules behind
		// the scenes
		settledResult, expiredResult, err := keepers.SettlePendingClaims(sdkCtx)
		require.NoError(t, err)
		require.Equal(t, 1, int(settledResult.NumClaims))
		require.Equal(t, 0, int(expiredResult.NumClaims))

		// Update the relay mining difficulty
		_, err = keepers.UpdateRelayMiningDifficulty(ctx, map[string]uint64{service.Id: numRelays})
		require.NoError(t, err)

		// Compute the expected reward
		expectedReward := numRelays * serviceComputeUnitsPerRelay * globalComputeUnitsToTokensMultiplier

		// Compute the new difficulty hash
		// newDifficultyHash := keepers.Keeper.ComputeNewDifficultyHash(ctx, numRelays)

		// // Check that the new difficulty hash is correct
		// require.Equal(t, expectedReward, newDifficultyHash.Reward)
	}

	// Create a tree + claim where:
	// - 1 Relay -> Earn small reward
	// - 10 Relays -> Increase reward
	// - 100 Relays -> Increase reward
	// - 1000,...,1e12 Relays -> Continue increasing reward

	// Update the relay mining difficulty and
	// - Check that EMA is changing
	// - Check that the difficulty is changing

	// Maintain a map of {num_relays -> num_rewards}
	// Then compute, for everything we have in the map (double list)
	// - Ratio of curr_relays to prev_relays
	// - Ratio of curr_rewards to prev_rewards
	// - Ensure the above are the same
}

// prepareSMST prepares an SMST with the given number of mined relays.
func prepareRealClaim(
	t *testing.T, ctx context.Context,
	numRelays uint64,
	service *sharedtypes.Service,
	app *apptypes.Application,
	supplier *sharedtypes.Supplier,
) prooftypes.Claim {
	t.Helper()

	// Generating an ephemeral tree & spec just so we can submit
	// a proof of the right size.
	// TODO_TECHDEBT(#446): Centralize the configuration for the SMT spec.
	kvStore, err := pebble.NewKVStore("")
	require.NoError(t, err)

	trie := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

	for i := uint64(0); i < numRelays; i++ {
		// DEV_NOTE: A signed mined relay is a MinedRelay type with the appropriate
		// payload, signatures and metadata populated.
		// It does not (as of writing) adhere to the actual on-chain difficulty (i.e.
		// hash check) of the test service surrounding the scope of this test.
		minedRelay := testrelayer.NewSignedMinedRelay(t, ctx,
			session,
			integrationApp.DefaultApplication.Address,
			integrationApp.DefaultSupplier.OperatorAddress,
			integrationApp.DefaultSupplierKeyringKeyringUid,
			integrationApp.GetKeyRing(),
			integrationApp.GetRingClient(),
		)

		err = trie.Update(minedRelay.Hash, minedRelay.Bytes, 1)
		require.NoError(t, err)
	}

	return trie
}
