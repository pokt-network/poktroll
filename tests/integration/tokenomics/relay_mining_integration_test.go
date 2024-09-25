package integration_test

import (
	"context"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var (
// TODO_BETA(#826): Uncomment these
// Test params.
// computeUnitsToTokensMultiplier = uint64(1) // keeping the math simple
// proofRequirementThreshold      = sdk.NewInt64Coin(volatile.DenomuPOKT, 1e18)
)

type RelayMiningIntegrationTestSuite struct {
	suites.ParamsSuite
}

func (s *RelayMiningIntegrationTestSuite) SetupTest() {
	// Construct a fresh integration app for each test.
	s.NewApp(s.T())
	s.SetupTestAuthzAccounts()
	s.SetupTestAuthzGrants()
}

func (s *RelayMiningIntegrationTestSuite) TestComputeNewDifficultyHash_RewardsReflectWorkCompleted() {
	// Set the shared module param compute_units_to_tokens_multiplier.
	// TODO_BETA(#826): wait for integration app & suites refactor to be merged.
	// _, err := s.RunUpdateParam(s.T(),
	// 	sharedtypes.ModuleName,
	// 	sharedtypes.ParamComputeUnitsToTokensMultiplier,
	// 	computeUnitsToTokensMultiplier,
	// )
	// require.NoError(s.T(), err)

	// Set the proof params so we never need a proof (for simplicity of this test)
	// TODO_BETA(#826): wait for integration app & suites refactor to be merged.
	// _, err = s.RunUpdateParam(s.T(),
	// 	prooftypes.ModuleName,
	// 	prooftypes.ParamProofRequestProbability,
	// 	float32(0),
	// )
	// require.NoError(s.T(), err)

	// Set the proof requirement threshold to be VERY high.
	// TODO_BETA(#826): wait for integration app & suites refactor to be merged.
	// _, err = s.RunUpdateParam(s.T(),
	// 	prooftypes.ModuleName,
	// 	prooftypes.ParamProofRequirementThreshold,
	// 	&proofRequirementThreshold,
	// )
	// require.NoError(s.T(), err)

	// TODO(@red-0ne, #781): Implement this test after the business logic is done.

	/*
		// Determine the height at which the claim will expire.
		sharedParams := sharedtypes.DefaultParams()
		claimWindowSizeBlocks := int64(sharedParams.GetClaimWindowOpenOffsetBlocks() + sharedParams.GetClaimWindowCloseOffsetBlocks())
		proofWindowSizeBlocks := int64(sharedParams.GetProofWindowOpenOffsetBlocks() + sharedParams.GetProofWindowCloseOffsetBlocks())

		app := integrationApp.DefaultApplication
		supplier := integrationApp.DefaultSupplier
		service := integrationApp.DefaultService

		// Monotonically increase the number of relays from a very small number
		// to a very large number
		for numRelays := uint64(1e3); numRelays <= 1e16; numRelays *= 10 {
			session := getSession(t, integrationApp)

			sessionEndHeight := session.GetHeader().GetSessionEndBlockHeight()
			claimExpirationHeight := int64(sessionEndHeight + claimWindowSizeBlocks + proofWindowSizeBlocks + 1)

			ctxAtHeight := sdkCtx.WithBlockHeight(claimExpirationHeight)

			relayMiningDifficulty, ok := keepers.TokenomicsKeeper.GetRelayMiningDifficulty(ctxAtHeight, service.Id)
			require.True(t, ok)

			// Prepare a claim with the given number of relays and store it
			claim := prepareRealClaim(t, ctxAtHeight, integrationApp, numRelays, app, supplier, session, service, &relayMiningDifficulty)
			keepers.ProofKeeper.UpsertClaim(ctxAtHeight, *claim)

			// Calling SettlePendingClaims calls ProcessTokenLogicModules behind the scenes
			settledResult, expiredResult, err := keepers.TokenomicsKeeper.SettlePendingClaims(ctxAtHeight)
			require.NoError(t, err)
			require.Equal(t, 1, int(settledResult.NumClaims))
			require.Equal(t, 0, int(expiredResult.NumClaims))

			// Update the relay mining difficulty
			_, err = keepers.TokenomicsKeeper.UpdateRelayMiningDifficulty(ctxAtHeight, map[string]uint64{service.Id: numRelays})
			require.NoError(t, err)

			// Compute the expected reward
			expectedReward := numRelays * serviceComputeUnitsPerRelay * computeUnitsToTokensMultiplier
			fmt.Println("Expected reward:", expectedReward)

			// Compute the new difficulty hash
			newDifficultyHash := protocol.ComputeNewDifficultyHash(ctx, numRelays)

			// // Check that the new difficulty hash is correct
			require.Equal(t, expectedReward, newDifficultyHash.Reward)

			// Update the relay mining difficulty and
			// - Check that EMA is changing
			// - Check that the difficulty is changing

			// Maintain a map of {num_relays -> num_rewards}
			// Then compute, for everything we have in the map (double list)
			// - Ratio of curr_relays to prev_relays
			// - Ratio of curr_rewards to prev_rewards
			// - Ensure the above are the same
		}
	*/
}

// prepareRealClaim prepares a claim by creating a real SMST with the given number
// of mined relays that adhere to the actual on-chain difficulty of the test service.
//
//nolint:unused // Will be used once the test above is implemented.
func prepareRealClaim(
	t *testing.T, ctx context.Context,
	integrationApp *integration.App,
	numRelays uint64,
	app *apptypes.Application,
	supplier *sharedtypes.Supplier,
	session *sessiontypes.Session,
	service *sharedtypes.Service,
	relayMiningDifficulty *tokenomicstypes.RelayMiningDifficulty,
) *prooftypes.Claim {
	t.Helper()

	// Prepare an in-memory key-value store
	kvStore, err := pebble.NewKVStore("")
	require.NoError(t, err)

	// Prepare an SMST
	trie := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

	// Insert the mined relays into the SMST
	for i := uint64(0); i < numRelays; i++ {
		// Mine a real relay
		minedRelay := testrelayer.NewSignedMinedRelay(t, ctx,
			session,
			app.Address,
			supplier.OperatorAddress,
			integrationApp.DefaultSupplierKeyringKeyringUid,
			integrationApp.GetKeyRing(),
			integrationApp.GetRingClient(),
		)
		// Ensure that the relay is applicable to the relay mining difficulty
		if protocol.IsRelayVolumeApplicable(minedRelay.Hash, relayMiningDifficulty.TargetHash) {
			err = trie.Update(minedRelay.Hash, minedRelay.Bytes, service.ComputeUnitsPerRelay)
			require.NoError(t, err)
		}
	}

	// Return the applicable claim
	return &prooftypes.Claim{
		SupplierOperatorAddress: integrationApp.DefaultSupplier.GetOperatorAddress(),
		SessionHeader:           session.GetHeader(),
		RootHash:                trie.Root(),
	}
}

func TestRelayMiningIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RelayMiningIntegrationTestSuite))
}
