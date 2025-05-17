package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParam_UpdateNumSuppliersPerSessionOnly(t *testing.T) {
	var expectedNumSuppliersPerSession uint64 = 420

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := sessiontypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedNumSuppliersPerSession, defaultParams.NumSuppliersPerSession)

	// Update the new parameter
	updateParamMsg := &sessiontypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sessiontypes.ParamNumSuppliersPerSession,
		AsType:    &sessiontypes.MsgUpdateParam_AsUint64{AsUint64: expectedNumSuppliersPerSession},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)

	// Assert that the onchain compute units to token multiplier is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedNumSuppliersPerSession, params.NumSuppliersPerSession)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSessionParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.NumSuppliersPerSession, params.NumSuppliersPerSession)
	require.Equal(t, expectedNumSuppliersPerSession, params.NumSuppliersPerSession)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(sessiontypes.KeyNumSuppliersPerSession))
}
