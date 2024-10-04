package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/testutil/integration"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func init() {
	cmd.InitSDKConfig()
}

// This is an example integration test @Olshansk was developing while implementing
// `testutil/integration/app.go` to test and verify different behaviors from
// setup, querying, running messages, etc...
// TODO_TECHDEBT: Once other integration tests exist or this test is refactored
// to be something more concrete and useful, decide if this should be deleted.
func TestTokenomicsIntegrationExample(t *testing.T) {
	// Create a new integration app
	integrationApp := integration.NewCompleteIntegrationApp(t)

	// Query and validate the default shared params
	sharedQueryClient := sharedtypes.NewQueryClient(integrationApp.QueryHelper())
	sharedParamsReq := sharedtypes.QueryParamsRequest{}
	sharedQueryRes, err := sharedQueryClient.Params(integrationApp.GetSdkCtx(), &sharedParamsReq)
	require.NoError(t, err)
	require.NotNil(t, sharedQueryRes, "unexpected nil params query response")
	require.EqualValues(t, sharedtypes.DefaultParams(), sharedQueryRes.GetParams())

	sharedParams := sharedQueryRes.GetParams()

	// Prepare a request to query a session so it can be used to create a claim.
	sessionQueryClient := sessiontypes.NewQueryClient(integrationApp.QueryHelper())
	getSessionReq := sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: integrationApp.DefaultApplication.Address,
		ServiceId:          integrationApp.DefaultService.Id,
		BlockHeight:        integrationApp.GetSdkCtx().BlockHeight(),
	}

	// Query the session
	getSessionRes, err := sessionQueryClient.GetSession(integrationApp.GetSdkCtx(), &getSessionReq)
	require.NoError(t, err)

	session := getSessionRes.GetSession()
	require.NotNil(t, session, "unexpected nil queryResponse")

	// Figure out how many blocks we need to wait until the earliest claim commit height
	// Query and validate the default shared params
	var claimWindowOpenBlockHash []byte
	earliestClaimCommitHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		session.GetHeader().GetSessionEndBlockHeight(),
		claimWindowOpenBlockHash,
		integrationApp.DefaultSupplier.GetOperatorAddress(),
	)

	// Need to wait until the earliest claim commit height
	currentBlockHeight := integrationApp.GetSdkCtx().BlockHeight()
	numBlocksUntilClaimWindowIsOpen := int(earliestClaimCommitHeight - currentBlockHeight)
	for i := 0; i < numBlocksUntilClaimWindowIsOpen; i++ {
		integrationApp.NextBlock(t)
	}

	// Create a new claim
	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierOperatorAddress: integrationApp.DefaultSupplier.GetOperatorAddress(),
		SessionHeader:           session.GetHeader(),
		RootHash:                testutilproof.SmstRootWithSumAndCount(1, 1),
	}

	// Run the message to create the claim
	result, err := integrationApp.RunMsg(t, &createClaimMsg)
	require.NoError(t, err)
	require.NotNil(t, result, "unexpected nil result")
}
