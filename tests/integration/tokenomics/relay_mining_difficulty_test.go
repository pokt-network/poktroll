package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	testutil "github.com/pokt-network/poktroll/testutil/integration"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
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
	for i := 0; i < 3; i++ {
		integrationApp.NextBlock(t)
	}

	// Get the current session and shared params
	session := getSession(t, integrationApp)
	sharedParams := getSharedParams(t, integrationApp)

	// Figure out how many blocks we need to wait until the claim window is open
	currentBlockHeight := int(integrationApp.SdkCtx().BlockHeight())
	sessionEndHeight := int(session.Header.SessionEndBlockHeight)
	claimOpenWindowNumBlocks := int(sharedParams.ClaimWindowOpenOffsetBlocks)
	numBlocksUntilClaimWindowIsOpen := int(sessionEndHeight + claimOpenWindowNumBlocks - currentBlockHeight + 1)

	// Wait until the claim window is open
	for i := 0; i < numBlocksUntilClaimWindowIsOpen; i++ {
		integrationApp.NextBlock(t)
	}

	// Create a new claim and create it
	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   session.Header,
		RootHash:        testutilproof.SmstRootWithSum(uint64(1)),
	}
	result := integrationApp.RunMsg(t,
		&createClaimMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result")

	// Figure out how many blocks we need to wait until the proof window is open
	currentBlockHeight = int(integrationApp.SdkCtx().BlockHeight())
	proofOpenWindowNumBlocks := int(sharedParams.ProofWindowOpenOffsetBlocks)
	claimCloseWindowNumBlocks := int(sharedParams.ClaimWindowCloseOffsetBlocks)
	numBlocksUntilProofWindowIsOpen := int(sessionEndHeight + claimOpenWindowNumBlocks + claimCloseWindowNumBlocks + proofOpenWindowNumBlocks - currentBlockHeight + 1)

	// Wait until the claim window is open
	for i := 0; i < numBlocksUntilProofWindowIsOpen; i++ {
		integrationApp.NextBlock(t)
	}

	// Create a new proof and submit it
	createProofMsg := prooftypes.MsgSubmitProof{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   session.Header,
		Proof:           []byte("abc"),
	}
	result = integrationApp.RunMsg(t,
		&createProofMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
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
