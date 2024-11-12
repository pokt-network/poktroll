package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestMsgUpdateParam_UpdateMintAllocationDaoOnly(t *testing.T) {
	var expectedMintAllocationDao float64 = 3.14159

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintAllocationDao, defaultParams.MintAllocationDao)

	// Update the new parameter
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintAllocationDao,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedMintAllocationDao},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)
	require.Equal(t, expectedMintAllocationDao, res.Params.MintAllocationDao)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(tokenomicstypes.KeyMintAllocationDao))
}
