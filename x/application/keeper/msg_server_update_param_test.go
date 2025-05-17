package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParam_UpdateMaxDelegatedGatewaysOnly(t *testing.T) {
	expectedMaxDelegatedGateways := uint64(999)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := apptypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMaxDelegatedGateways, defaultParams.MaxDelegatedGateways)

	// Update the max delegated gateways
	updateParamMsg := &apptypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      apptypes.ParamMaxDelegatedGateways,
		AsType:    &apptypes.MsgUpdateParam_AsUint64{AsUint64: expectedMaxDelegatedGateways},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain max delegated gateways is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedMaxDelegatedGateways, params.MaxDelegatedGateways)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateApplicationParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MaxDelegatedGateways, params.MaxDelegatedGateways)
	require.Equal(t, expectedMaxDelegatedGateways, params.MaxDelegatedGateways)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(apptypes.KeyMaxDelegatedGateways))
}

func TestMsgUpdateParam_UpdateMinStakeOnly(t *testing.T) {
	expectedMinStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 420)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := apptypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMinStake, defaultParams.MinStake)

	// Update the application min stake
	updateParamMsg := &apptypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      apptypes.ParamMinStake,
		AsType:    &apptypes.MsgUpdateParam_AsCoin{AsCoin: &expectedMinStake},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain application min stake is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedMinStake, params.MinStake)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateApplicationParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MinStake, params.MinStake)
	require.Equal(t, &expectedMinStake, params.MinStake)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(apptypes.KeyMinStake))
}
