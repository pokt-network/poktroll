package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/app/volatile"
	testkeeper "github.com/pokt-network/pocket/testutil/keeper"
	apptypes "github.com/pokt-network/pocket/x/application/types"
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
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.MaxDelegatedGateways, res.Params.MaxDelegatedGateways)
	require.Equal(t, expectedMaxDelegatedGateways, res.Params.MaxDelegatedGateways)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(apptypes.KeyMaxDelegatedGateways))
}

func TestMsgUpdateParam_UpdateMinStakeOnly(t *testing.T) {
	expectedMinStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 420)

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
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.MinStake, res.Params.MinStake)
	require.Equal(t, expectedMinStake.Amount, res.Params.MinStake.Amount)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(apptypes.KeyMinStake))
}
