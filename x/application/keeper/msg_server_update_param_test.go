package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func TestMsgUpdateParam_UpdateMaxDelegatedGatewaysOnly(t *testing.T) {
	expectedMaxDelegatedGateways := uint64(999)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := apptypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

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

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MaxDelegatedGateways, updatedParams.MaxDelegatedGateways)
	require.Equal(t, expectedMaxDelegatedGateways, updatedParams.MaxDelegatedGateways)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(apptypes.KeyMaxDelegatedGateways))
}

func TestMsgUpdateParam_UpdateMinStakeOnly(t *testing.T) {
	expectedMinStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 420)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := apptypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMinStake, defaultParams.MinStake)

	// Update the min relay difficulty bits
	updateParamMsg := &apptypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      apptypes.ParamMinStake,
		AsType:    &apptypes.MsgUpdateParam_AsCoin{AsCoin: &expectedMinStake},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MinStake, updatedParams.MinStake)
	require.Equal(t, expectedMinStake.Amount, updatedParams.MinStake.Amount)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(apptypes.KeyMinStake))
}
