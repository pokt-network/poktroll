package integration_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	"cosmossdk.io/store/rootmulti"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/integration"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestSessionMemo_ClaimsThroughFinalizeBlock drives MsgCreateClaim through the
// integration app's real FinalizeBlock (ExecModeFinalize, tx store branch) and
// Commit, so the block-scoped session memo is exercised end-to-end: claims for
// the same session in one tx succeed, and nothing survives the block.
func TestSessionMemo_ClaimsThroughFinalizeBlock(t *testing.T) {
	integrationApp := integration.NewCompleteIntegrationApp(t)

	sharedParams := sharedtypes.DefaultParams()
	sessionQueryClient := sessiontypes.NewQueryClient(integrationApp.QueryHelper())
	getSessionRes, err := sessionQueryClient.GetSession(integrationApp.GetSdkCtx(), &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: integrationApp.DefaultApplication.Address,
		ServiceId:          integrationApp.DefaultService.Id,
		BlockHeight:        integrationApp.GetSdkCtx().BlockHeight(),
	})
	require.NoError(t, err)
	session := getSessionRes.GetSession()

	// Advance to the claim window close height so the claim is accepted at any
	// supplier-specific commit height.
	claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, session.GetHeader().GetSessionEndBlockHeight())
	for integrationApp.GetSdkCtx().BlockHeight() < claimWindowCloseHeight {
		integrationApp.NextBlock(t)
	}

	// Two claims for the same session in one tx: the second is a memo hit.
	// (Same supplier twice is an upsert, which keeps the fixture to one supplier.)
	createClaimMsg := &prooftypes.MsgCreateClaim{
		SupplierOperatorAddress: integrationApp.DefaultSupplier.GetOperatorAddress(),
		SessionHeader:           session.GetHeader(),
		RootHash:                testutilproof.SmstRootWithSumAndCount(1, 1),
	}
	// RunMsgs reports one response per tx; an error means any msg in it failed.
	msgResps, err := integrationApp.RunMsgs(t, createClaimMsg, createClaimMsg)
	require.NoError(t, err)
	require.Len(t, msgResps, 1)

	// The block is committed: the memo written during FinalizeBlock is gone.
	rootStore, ok := integrationApp.CommitMultiStore().(*rootmulti.Store)
	require.True(t, ok)
	transientKey := rootStore.StoreKeysByName()[sessiontypes.TransientStoreKey]
	require.NotNil(t, transientKey, "session transient store is not mounted on the integration app")

	memoIterator := prefix.NewStore(rootStore.GetKVStore(transientKey), sessiontypes.SessionMemoKeyPrefix).Iterator(nil, nil)
	defer memoIterator.Close()
	require.False(t, memoIterator.Valid(), "session memo survived Commit")

	// The claim itself persisted.
	proofQueryClient := prooftypes.NewQueryClient(integrationApp.QueryHelper())
	claimRes, err := proofQueryClient.Claim(integrationApp.GetSdkCtx(), &prooftypes.QueryGetClaimRequest{
		SessionId:               session.GetSessionId(),
		SupplierOperatorAddress: integrationApp.DefaultSupplier.GetOperatorAddress(),
	})
	require.NoError(t, err)
	claim := claimRes.GetClaim()
	require.Equal(t, session.GetSessionId(), claim.GetSessionHeader().GetSessionId())
}
