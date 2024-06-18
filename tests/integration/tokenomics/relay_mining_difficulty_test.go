package integration_test

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/badger"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	testutil "github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_UPNEXT(@Olshansk, #571): Implement these tests

func init() {
	cmd.InitSDKConfig()
}

func TestUpdateRelayMiningDifficulty_NewServiceSeenForTheFirstTime(t *testing.T) {
	// Create a new integration app
	integrationApp := integration.NewCompleteIntegrationApp(t)

	// Move forward a few blocks to move away from the genesis block
	integrationApp.NextBlocks(t, 3)

	// Get the current session and shared params
	session := getSession(t, integrationApp)
	sharedParams := getSharedParams(t, integrationApp)

	// Prepare the trie with a single mined relay
	trie := prepareSMST(t, integrationApp.SdkCtx(), integrationApp, session)

	// Compute the number of blocks to wait between different events
	currentBlockHeight := int(integrationApp.SdkCtx().BlockHeight())
	sessionEndHeight := int(session.Header.SessionEndBlockHeight)
	claimOpenWindowNumBlocks := int(sharedParams.ClaimWindowOpenOffsetBlocks)
	claimCloseWindowNumBlocks := int(sharedParams.ClaimWindowCloseOffsetBlocks)
	proofOpenWindowNumBlocks := int(sharedParams.ProofWindowOpenOffsetBlocks)
	proofCloseWindowNumBlocks := int(sharedParams.ProofWindowCloseOffsetBlocks)
	numBlocksUntilClaimWindowIsOpen := int(sessionEndHeight + claimOpenWindowNumBlocks - currentBlockHeight + 1)
	numBlocksUntilProofWindowIsOpen := numBlocksUntilClaimWindowIsOpen + claimCloseWindowNumBlocks + proofOpenWindowNumBlocks
	numBlocksUntilProofWindowIsClosed := numBlocksUntilProofWindowIsOpen + proofCloseWindowNumBlocks

	// Wait until the claim window is open
	integrationApp.NextBlocks(t, numBlocksUntilClaimWindowIsOpen)

	// Create a new claim and create it
	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   session.Header,
		RootHash:        trie.Root(),
	}
	result := integrationApp.RunMsg(t,
		&createClaimMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result when submitting a MsgCreateClaim tx")

	// Wait until the proof window is open
	integrationApp.NextBlocks(t, numBlocksUntilProofWindowIsOpen)

	// Create a new proof and submit it
	createProofMsg := prooftypes.MsgSubmitProof{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   session.Header,
		Proof:           getProof(t, trie),
	}
	result = integrationApp.RunMsg(t,
		&createProofMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result when submitting a MsgSubmitProof tx")

	// Wait until the proof window is closed
	integrationApp.NextBlocks(t, numBlocksUntilProofWindowIsClosed)

}

func UpdateRelayMiningDifficulty_UpdatingMultipleServicesAtOnce(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdateServiceIsNotSeenForAWhile(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdateServiceIsIncreasing(t *testing.T) {}

func UpdateRelayMiningDifficulty_UpdateServiceIsDecreasing(t *testing.T) {}

func getSharedParams(t *testing.T, integrationApp *testutil.App) sharedtypes.Params {
	t.Helper()

	sharedQueryClient := sharedtypes.NewQueryClient(integrationApp.QueryHelper())
	sharedParamsReq := sharedtypes.QueryParamsRequest{}

	sharedQueryRes, err := sharedQueryClient.Params(integrationApp.SdkCtx(), &sharedParamsReq)
	require.NoError(t, err)

	return sharedQueryRes.Params
}

func getSession(t *testing.T, integrationApp *testutil.App) *sessiontypes.Session {
	t.Helper()

	sessionQueryClient := sessiontypes.NewQueryClient(integrationApp.QueryHelper())
	getSessionReq := sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: integrationApp.DefaultApplication.Address,
		Service:            integrationApp.DefaultService,
		BlockHeight:        integrationApp.SdkCtx().BlockHeight(),
	}

	getSessionRes, err := sessionQueryClient.GetSession(integrationApp.SdkCtx(), &getSessionReq)
	require.NoError(t, err)
	require.NotNil(t, getSessionRes, "unexpected nil queryResponse")
	return getSessionRes.Session
}

// prepareSMST prepares an SMST with a single mined relay for the given session.
func prepareSMST(
	t *testing.T, ctx context.Context,
	integrationApp *testutil.App,
	session *sessiontypes.Session,
) *smt.SMST {
	t.Helper()

	// Generating an ephemeral tree & spec just so we can submit
	// a proof of the right size.
	// TODO_TECHDEBT(#446): Centralize the configuration for the SMT spec.
	kvStore, err := badger.NewKVStore("")
	require.NoError(t, err)

	minedRelay := testrelayer.NewSignedMinedRelay(t, ctx,
		session,
		integrationApp.DefaultApplication.Address,
		integrationApp.DefaultSupplier.Address,
		integrationApp.DefaultSupplierKeyringKeyringUid,
		integrationApp.KeyRing(),
		integrationApp.RingClient(),
	)

	trie := smt.NewSparseMerkleSumTrie(kvStore, sha256.New(), smt.WithValueHasher(nil))
	err = trie.Update(minedRelay.Hash, minedRelay.Bytes, 1)
	require.NoError(t, err)

	return trie
}

// getProof returns a proof for the given session.
func getProof(t *testing.T, trie *smt.SMST) []byte {
	t.Helper()

	emptyPath := make([]byte, trie.PathHasherSize())
	proof, err := trie.ProveClosest(emptyPath)
	require.NoError(t, err)

	proofBz, err := proof.Marshal()
	require.NoError(t, err)

	return proofBz
}
