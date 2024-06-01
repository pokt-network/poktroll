package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

func TestRelayMiningDifficulty(t *testing.T) {

	params := tokenomicstypes.DefaultParams()
	params.ComputeUnitsToTokensMultiplier = 42

	app := integration.NewCompleteIntegrationApp(t)

	req := &tokenomicstypes.MsgUpdateParam{
		Authority: app.Authority.String(),
		Name:      "compute_units_to_tokens_multiplier",
		AsType:    &tokenomicstypes.MsgUpdateParam_AsInt64{AsInt64: 10},
	}

	result, err := app.RunMsg(
		req,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NoError(t, err)
	require.NotNil(t, result, "unexpected nil result")

	// we now check the result
	resp := tokenomicstypes.MsgUpdateParamResponse{}
	err = app.Cdc.Unmarshal(result.Value, &resp)
	require.NoError(t, err)

	// we should also check the state of the application
	// gotParams := tokenomicsKeeper.GetParams(app.Ctx)
	// require.NotNil(t, gotParams, "unexpected nil params")
	// fmt.Println(gotParams.ComputeUnitsToTokensMultiplier) // Output: 10000

	// query := tokenomicstypes.QueryAllRelayMiningDifficultyRequest
	// app.QueryHelper()
}
