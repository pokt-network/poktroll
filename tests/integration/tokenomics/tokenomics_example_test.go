package integration_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
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

	// Prepare a request to update the compute_units_to_tokens_multiplier
	req := &tokenomicstypes.MsgUpdateParam{
		Authority: integrationApp.Authority.String(),
		Name:      "compute_units_to_tokens_multiplier",
		AsType:    &tokenomicstypes.MsgUpdateParam_AsInt64{AsInt64: 10},
	}

	// Run the request
	result := integrationApp.RunMsg(t,
		req,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NotNil(t, result, "unexpected nil result")

	// Validate the response
	resp := tokenomicstypes.MsgUpdateParamResponse{}
	err := integrationApp.Cdc.Unmarshal(result.Value, &resp)
	require.NoError(t, err)

	// Create a query client
	queryClient := tokenomicstypes.NewQueryClient(integrationApp.QueryHelper)

	// Query the updated value
	queryParams := tokenomicstypes.QueryParamsRequest{}
	queryResponse, err := queryClient.Params(integrationApp.Ctx, &queryParams)
	require.NoError(t, err)
	require.NotNil(t, queryResponse, "unexpected nil queryResponse")
	require.Equal(t, uint64(10), queryResponse.Params.ComputeUnitsToTokensMultiplier)

	// Prepare a new supplier
	supplierStake := types.NewCoin("upokt", math.NewInt(1000000))
	supplier := sharedtypes.Supplier{
		Address: sample.AccAddress(),
		Stake:   &supplierStake,
	}

	// Prepare a new application
	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &appStake,
	}

	sessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress: app.Address,
		Service: &sharedtypes.Service{
			Id:   "svc1",
			Name: "svcName1",
		},
		SessionId:               "session_id",
		SessionStartBlockHeight: 1,
		SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
	}
	claim := prooftypes.Claim{
		SupplierAddress: supplier.Address,
		SessionHeader:   sessionHeader,
		RootHash:        testproof.SmstRootWithSum(appStake.Amount.Uint64() + 1), // More than the app stake
	}

	createClaimMsg := prooftypes.MsgCreateClaim{
		SupplierAddress: supplier.Address,
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