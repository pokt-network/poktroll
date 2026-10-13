package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgUpdateParam_UpdateMinStakeOnly(t *testing.T) {
	expectedMinStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 420)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := suppliertypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMinStake, defaultParams.MinStake)

	// Update the min relay difficulty bits
	updateParamMsg := &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamMinStake,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &expectedMinStake},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MinStake, updatedParams.MinStake)
	require.Equal(t, expectedMinStake.Amount, updatedParams.MinStake.Amount)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(suppliertypes.KeyMinStake))
}

func TestMsgUpdateParam_UpdateMinStakeIncludedServices(t *testing.T) {
	k, msgSrv, ctx := setupMsgServer(t)
	require.NoError(t, k.SetParams(ctx, suppliertypes.DefaultParams()))

	_, err := msgSrv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamMinStakeIncludedServices,
		AsType:    &suppliertypes.MsgUpdateParam_AsUint64{AsUint64: 35},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(35), k.GetParams(ctx).MinStakeIncludedServices)

	_, err = msgSrv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamMinStakeIncludedServices,
		AsType:    &suppliertypes.MsgUpdateParam_AsUint64{AsUint64: 0},
	})
	require.Error(t, err)
	require.Equal(t, uint64(35), k.GetParams(ctx).MinStakeIncludedServices)
}

func TestMsgUpdateParam_UpdateStakePerAdditionalService(t *testing.T) {
	k, msgSrv, ctx := setupMsgServer(t)
	require.NoError(t, k.SetParams(ctx, suppliertypes.DefaultParams()))

	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 5_000_000_000)
	_, err := msgSrv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamStakePerAdditionalService,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &price},
	})
	require.NoError(t, err)
	require.Equal(t, price.Amount, k.GetParams(ctx).StakePerAdditionalService.Amount)
}
