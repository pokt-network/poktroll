package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

func TestTokenomicsExample(t *testing.T) {
	integrationApp := integration.NewCompleteIntegrationApp(t)

	// Query shared params updated value
	sharedQueryClient := sharedtypes.NewQueryClient(integrationApp.QueryHelper())
	sharedQueryParams := sharedtypes.QueryParamsRequest{}
	sharedQueryResponse, err := sharedQueryClient.Params(integrationApp.SdkCtx(), &sharedQueryParams)
	require.NoError(t, err)
	require.NotNil(t, sharedQueryResponse, "unexpected nil queryResponse")
	require.EqualValues(t, uint64(4), sharedQueryResponse.Params.NumBlocksPerSession)

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

	// Validate the response
	resp := tokenomicstypes.MsgUpdateParamResponse{}
	err = integrationApp.Codec().Unmarshal(result.Value, &resp)
	require.NoError(t, err)

	// Create a query client
	tokenomicsQueryClient := tokenomicstypes.NewQueryClient(integrationApp.QueryHelper())

	// Query the updated value
	tokenomicsQueryParams := tokenomicstypes.QueryParamsRequest{}
	tokenomicsQueryResponse, err := tokenomicsQueryClient.Params(integrationApp.SdkCtx(), &tokenomicsQueryParams)
	require.NoError(t, err)
	require.NotNil(t, tokenomicsQueryResponse, "unexpected nil queryResponse")
	require.EqualValues(t, uint64(11), uint64(tokenomicsQueryResponse.Params.ComputeUnitsToTokensMultiplier))

	sessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress: integrationApp.DefaultApplication.Address,
		Service: &sharedtypes.Service{
			Id:   "svc1",
			Name: "svcName1",
		},
		SessionId:               "session_id",
		SessionStartBlockHeight: 1,
		SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
	}

	claim := prooftypes.Claim{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   sessionHeader,
		RootHash:        testproof.SmstRootWithSum(uint64(1)),
	}

	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierAddress: integrationApp.DefaultSupplier.Address,
		SessionHeader:   sessionHeader,
		RootHash:        claim.RootHash,
	}

	integrationApp.NextBlock(t)

	result = integrationApp.RunMsg(t,
		&createClaimMsg,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result")
}
