package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

// This is an example integration test @Olshansk was developing while implementing
// `testutil/integration/app.go` to test and verify different behaviours from
// setup, querying, running messages, etc...
// TODO_TECHDEBT: Once other integration tests exist or this test is refactored
// to be something more concrete and useful, decide if this should be deleted.
func TestTokenomicsIntegrationExample(t *testing.T) {
	// Create a new integration app
	integrationApp := integration.NewCompleteIntegrationApp(t)

	// Query and validate the default shared params
	sharedQueryClient := sharedtypes.NewQueryClient(integrationApp.QueryHelper())
	sharedParamsReq := sharedtypes.QueryParamsRequest{}
	sharedQueryRes, err := sharedQueryClient.Params(integrationApp.SdkCtx(), &sharedParamsReq)
	require.NoError(t, err)
	require.NotNil(t, sharedQueryRes, "unexpected nil params query response")
	require.EqualValues(t, sharedtypes.DefaultParams(), sharedQueryRes.GetParams())

	// Prepare a request to update the compute_units_to_tokens_multiplier
	updateTokenomicsParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: integrationApp.Authority(),
		Name:      "compute_units_to_tokens_multiplier",
		AsType:    &tokenomicstypes.MsgUpdateParam_AsInt64{AsInt64: 11},
	}

	// Run the request
	result := integrationApp.RunMsg(t,
		updateTokenomicsParamMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result")

	// Validate the response is correct and that the value was updated
	resp := tokenomicstypes.MsgUpdateParamResponse{}
	err = integrationApp.Codec().Unmarshal(result.Value, &resp)
	require.NoError(t, err)

	tokenomicsQueryClient := tokenomicstypes.NewQueryClient(integrationApp.QueryHelper())

	tokenomicsQueryParams := tokenomicstypes.QueryParamsRequest{}
	tokenomicsQueryResponse, err := tokenomicsQueryClient.Params(integrationApp.SdkCtx(), &tokenomicsQueryParams)
	require.NoError(t, err)
	require.NotNil(t, tokenomicsQueryResponse, "unexpected nil queryResponse")
	require.EqualValues(t, uint64(11), uint64(tokenomicsQueryResponse.Params.ComputeUnitsToTokensMultiplier))

	// Commit & finalize the current block, then moving to the next one.
	integrationApp.NextBlock(t)

	// Prepare a request to query a session so it can be used to create a claim.
	sessionQueryClient := sessiontypes.NewQueryClient(integrationApp.QueryHelper())
	getSessionReq := sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: integrationApp.DefaultApplication.Address,
		Service:            integrationApp.DefaultService,
		BlockHeight:        3,
	}
	// Query the session
	getSessionRes, err := sessionQueryClient.GetSession(integrationApp.SdkCtx(), &getSessionReq)
	require.NoError(t, err)
	require.NotNil(t, getSessionRes, "unexpected nil queryResponse")

	// Create a new claim
	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   getSessionRes.Session.Header,
		RootHash:        testutilproof.SmstRootWithSum(uint64(1)),
	}

	// Run the message to create the claim
	result = integrationApp.RunMsg(t,
		&createClaimMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result")
}
