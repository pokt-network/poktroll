package integration_test

import (
	"context"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/integration"
	testutil "github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func init() {
	cmd.InitSDKConfig()
}

func TestUpdateRelayMiningDifficulty_NewServiceSeenForTheFirstTime(t *testing.T) {
	var claimWindowOpenBlockHash, proofWindowOpenBlockHash []byte

	// Create a new integration app
	integrationApp := integration.NewCompleteIntegrationApp(t)
	sdkCtx := integrationApp.GetSdkCtx()

	// Move forward a few blocks to move away from the genesis block
	integrationApp.NextBlocks(t, 3)

	// Get the current session and shared params
	session := getSession(t, integrationApp)
	sharedParams := getSharedParams(t, integrationApp)
	proofParams := getProofParams(t, integrationApp)

	// Update the proof parameters to never require a proof, since this test is not
	// submitting any proofs.
	maxProofRequirementThreshold := sdk.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &maxProofRequirementThreshold
	proofParams.ProofRequestProbability = 0

	msgProofParams := prooftypes.MsgUpdateParams{
		Authority: integrationApp.GetAuthority(),
		Params:    proofParams,
	}
	_, err := integrationApp.RunMsg(t, &msgProofParams)
	require.NoError(t, err)

	// Prepare the trie with several mined relays
	expectedNumRelays := uint64(100)
	trie := prepareSMST(t, sdkCtx, integrationApp, session, expectedNumRelays)

	// Compute the number of blocks to wait between different events
	// TODO_BLOCKER(@bryanchriswhite): See this comment: https://github.com/pokt-network/poktroll/pull/610#discussion_r1645777322
	sessionEndHeight := session.Header.SessionEndBlockHeight
	earliestSupplierClaimCommitHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionEndHeight,
		claimWindowOpenBlockHash,
		integrationApp.DefaultSupplier.GetOperatorAddress(),
	)
	earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionEndHeight,
		proofWindowOpenBlockHash,
		integrationApp.DefaultSupplier.GetOperatorAddress(),
	)
	proofWindowCloseHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)

	// Wait until the earliest claim commit height.
	currentBlockHeight := sdkCtx.BlockHeight()
	numBlocksUntilClaimWindowOpenHeight := earliestSupplierClaimCommitHeight - currentBlockHeight
	require.Greater(t, numBlocksUntilClaimWindowOpenHeight, int64(0), "unexpected non-positive number of blocks until the earliest claim commit height")
	integrationApp.NextBlocks(t, int(numBlocksUntilClaimWindowOpenHeight))

	// Construct a new create claim message and commit it.
	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierOperatorAddress: integrationApp.DefaultSupplier.OperatorAddress,
		SessionHeader:           session.Header,
		RootHash:                trie.Root(),
	}
	result, err := integrationApp.RunMsg(t, &createClaimMsg)
	require.NoError(t, err)
	require.NotNil(t, result, "unexpected nil result when submitting a MsgCreateClaim tx")

	// Wait until the proof window is open
	currentBlockHeight = sdkCtx.BlockHeight()
	numBlocksUntilProofWindowOpenHeight := earliestSupplierProofCommitHeight - currentBlockHeight
	require.Greater(t, numBlocksUntilProofWindowOpenHeight, int64(0), "unexpected non-positive number of blocks until the earliest proof commit height")
	integrationApp.NextBlocks(t, int(numBlocksUntilProofWindowOpenHeight))

	// Wait until the proof window is closed
	currentBlockHeight = sdkCtx.BlockHeight()
	numBlocksUntilProofWindowCloseHeight := proofWindowCloseHeight - currentBlockHeight
	require.Greater(t, numBlocksUntilProofWindowOpenHeight, int64(0), "unexpected non-positive number of blocks until the earliest proof commit height")

	// TODO_TECHDEBT(@bryanchriswhite): Olshansky is unsure why the +1 is necessary here but it was required to pass the test.
	integrationApp.NextBlocks(t, int(numBlocksUntilProofWindowCloseHeight)+1)

	// Check that the expected events are emitted
	events := sdkCtx.EventManager().Events()
	relayMiningEvents := testutilevents.FilterEvents[*servicetypes.EventRelayMiningDifficultyUpdated](t, events)
	require.Len(t, relayMiningEvents, 1, "unexpected number of relay mining difficulty updated events")
	relayMiningEvent := relayMiningEvents[0]
	require.Equal(t, "svc1", relayMiningEvent.ServiceId)

	// The default difficulty
	require.Equal(t, protocol.BaseRelayDifficultyHashHex, relayMiningEvent.PrevTargetHashHexEncoded)
	require.Equal(t, protocol.BaseRelayDifficultyHashHex, relayMiningEvent.NewTargetHashHexEncoded)

	// The previous EMA is the same as the current one if the service is new
	require.Equal(t, expectedNumRelays, relayMiningEvent.PrevNumRelaysEma)
	require.Equal(t, expectedNumRelays, relayMiningEvent.NewNumRelaysEma)
}

func UpdateRelayMiningDifficulty_UpdatingMultipleServicesAtOnce(t *testing.T) {
	t.Skip("TODO_TEST: Implement this test")
}

func UpdateRelayMiningDifficulty_UpdateServiceIsNotSeenForAWhile(t *testing.T) {
	t.Skip("TODO_TEST: Implement this test")
}

func UpdateRelayMiningDifficulty_UpdateServiceIsIncreasing(t *testing.T) {
	t.Skip("TODO_TEST: Implement this test")
}

func UpdateRelayMiningDifficulty_UpdateServiceIsDecreasing(t *testing.T) {
	t.Skip("TODO_TEST: Implement this test")
}

// getSharedParams returns the shared parameters for the current block height.
func getSharedParams(t *testing.T, integrationApp *testutil.App) sharedtypes.Params {
	t.Helper()

	sdkCtx := integrationApp.GetSdkCtx()

	sharedQueryClient := sharedtypes.NewQueryClient(integrationApp.QueryHelper())
	sharedParamsReq := sharedtypes.QueryParamsRequest{}

	sharedQueryRes, err := sharedQueryClient.Params(sdkCtx, &sharedParamsReq)
	require.NoError(t, err)

	return sharedQueryRes.Params
}

// getProofParams returns the proof parameters for the current block height.
func getProofParams(t *testing.T, integrationApp *testutil.App) prooftypes.Params {
	t.Helper()

	sdkCtx := integrationApp.GetSdkCtx()

	proofQueryClient := prooftypes.NewQueryClient(integrationApp.QueryHelper())
	proofParamsReq := prooftypes.QueryParamsRequest{}

	proofQueryRes, err := proofQueryClient.Params(sdkCtx, &proofParamsReq)
	require.NoError(t, err)

	return proofQueryRes.Params
}

// getSession returns the current session for the default application and service.
func getSession(t *testing.T, integrationApp *testutil.App) *sessiontypes.Session {
	t.Helper()

	sdkCtx := integrationApp.GetSdkCtx()

	sessionQueryClient := sessiontypes.NewQueryClient(integrationApp.QueryHelper())
	getSessionReq := sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: integrationApp.DefaultApplication.Address,
		ServiceId:          integrationApp.DefaultService.Id,
		BlockHeight:        sdkCtx.BlockHeight(),
	}

	getSessionRes, err := sessionQueryClient.GetSession(sdkCtx, &getSessionReq)
	require.NoError(t, err)
	require.NotNil(t, getSessionRes, "unexpected nil queryResponse")
	return getSessionRes.Session
}

// prepareSMST prepares an SMST with the given number of mined relays.
func prepareSMST(
	t *testing.T, ctx context.Context,
	integrationApp *testutil.App,
	session *sessiontypes.Session,
	numRelays uint64,
) *smt.SMST {
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
