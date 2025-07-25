package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

func TestMsgUpdateParam_UpdateAddServiceFeeOnly(t *testing.T) {
	expectedAddServiceFee := &sdk.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(1000000001)}

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := servicetypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedAddServiceFee, defaultParams.AddServiceFee)

	// Update the add service fee parameter
	updateParamMsg := &servicetypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      servicetypes.ParamAddServiceFee,
		AsType:    &servicetypes.MsgUpdateParam_AsCoin{AsCoin: expectedAddServiceFee},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	params := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.AddServiceFee, params.AddServiceFee)
	require.Equal(t, expectedAddServiceFee, params.AddServiceFee)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, "AddServiceFee")
}

func TestMsgUpdateParam_UpdateTargetNumRelaysOnly(t *testing.T) {
	expectedTargetNumRelays := uint64(9001)

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := servicetypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedTargetNumRelays, defaultParams.TargetNumRelays)

	// Update the add service fee parameter
	updateParamMsg := &servicetypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      servicetypes.ParamTargetNumRelays,
		AsType:    &servicetypes.MsgUpdateParam_AsUint64{AsUint64: expectedTargetNumRelays},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	params := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.TargetNumRelays, params.TargetNumRelays)
	require.Equal(t, expectedTargetNumRelays, params.TargetNumRelays)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, "TargetNumRelays")
}
