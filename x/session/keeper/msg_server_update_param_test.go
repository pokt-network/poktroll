package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/pocket/testutil/keeper"
	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

func TestMsgUpdateParam_UpdateNumSuppliersPerSessionOnly(t *testing.T) {
	var expectedNumSuppliersPerSession uint64 = 420

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := sessiontypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedNumSuppliersPerSession, defaultParams.NumSuppliersPerSession)

	// Update the new parameter
	updateParamMsg := &sessiontypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sessiontypes.ParamNumSuppliersPerSession,
		AsType:    &sessiontypes.MsgUpdateParam_AsUint64{AsUint64: expectedNumSuppliersPerSession},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)
	require.Equal(t, expectedNumSuppliersPerSession, res.Params.NumSuppliersPerSession)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(sessiontypes.KeyNumSuppliersPerSession))
}
