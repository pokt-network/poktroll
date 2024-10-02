package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgUpdateParam_UpdateMinStakeOnly(t *testing.T) {
	expectedMinStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 420)

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
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.MinStake, res.Params.MinStake)
	require.Equal(t, expectedMinStake.Amount, res.Params.MinStake.Amount)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, suppliertypes.ParamMinStake)
}
