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

func TestMsgUpdateParam_UpdateMintAllocationProposerOnly(t *testing.T) {
	var expectedMintAllocationProposer float64 = 3.14159

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintAllocationProposer, defaultParams.MintAllocationProposer)

	// Update the new parameter
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintAllocationProposer,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedMintAllocationProposer},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)
	require.Equal(t, expectedMintAllocationProposer, res.Params.MintAllocationProposer)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(tokenomicstypes.KeyMintAllocationProposer))
}

func TestMsgUpdateParam_UpdateMintAllocationSupplierOnly(t *testing.T) {
	var expectedMintAllocationSupplier float64 = 3.14159

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintAllocationSupplier, defaultParams.MintAllocationSupplier)

	// Update the new parameter
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintAllocationSupplier,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedMintAllocationSupplier},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)
	require.Equal(t, expectedMintAllocationSupplier, res.Params.MintAllocationSupplier)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(tokenomicstypes.KeyMintAllocationSupplier))
}

func TestMsgUpdateParam_UpdateMintAllocationSourceOwnerOnly(t *testing.T) {
	var expectedMintAllocationSourceOwner float64 = 3.14159

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintAllocationSourceOwner, defaultParams.MintAllocationSourceOwner)

	// Update the new parameter
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintAllocationSourceOwner,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedMintAllocationSourceOwner},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)
	require.Equal(t, expectedMintAllocationSourceOwner, res.Params.MintAllocationSourceOwner)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(tokenomicstypes.KeyMintAllocationSourceOwner))
}

func TestMsgUpdateParam_UpdateMintAllocationApplicationOnly(t *testing.T) {
	var expectedMintAllocationApplication float64 = 3.14159

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintAllocationApplication, defaultParams.MintAllocationApplication)

	// Update the new parameter
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintAllocationApplication,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedMintAllocationApplication},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)
	require.Equal(t, expectedMintAllocationApplication, res.Params.MintAllocationApplication)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(tokenomicstypes.KeyMintAllocationApplication))
}
