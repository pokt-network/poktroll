package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParam_UpdateMinStakeOnly(t *testing.T) {
	expectedMinStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 420)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := gatewaytypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMinStake, defaultParams.MinStake)

	// Update the gateway min stake
	updateParamMsg := &gatewaytypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      gatewaytypes.ParamMinStake,
		AsType:    &gatewaytypes.MsgUpdateParam_AsCoin{AsCoin: &expectedMinStake},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)

	// Assert that the onchain gateway min stake is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedMinStake, params.MinStake)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateGatewayParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, expectedMinStake, params.MinStake)
	require.Equal(t, &expectedMinStake, params.MinStake)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(gatewaytypes.KeyMinStake))
}
