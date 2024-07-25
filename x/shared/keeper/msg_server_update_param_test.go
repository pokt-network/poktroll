package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/shared/keeper"
	"github.com/pokt-network/poktroll/x/shared/types"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParam_UpdateNumBlocksPerSession(t *testing.T) {
	var expectedNumBlocksPerSession int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedNumBlocksPerSession), defaultParams.NumBlocksPerSession)

	// Update the number of blocks per session
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedNumBlocksPerSession},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedNumBlocksPerSession), res.Params.NumBlocksPerSession)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "NumBlocksPerSession")
}

func TestMsgUpdateParam_UpdateClaimWindowOpenOffsetBlocks(t *testing.T) {
	var expectedClaimWindowOpenOffestBlocks int64 = 4

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	defaultParams := sharedtypes.DefaultParams()

	// Get the delta between the previous claim window open offset blocks and its
	// new value to update the supplier unbonding period such as it is greater than
	// the cumulative proof window close blocks to pass UpdateParam validation.
	paramDelta := uint64(expectedClaimWindowOpenOffestBlocks) - defaultParams.ClaimWindowOpenOffsetBlocks
	defaultParams.SupplierUnbondingPeriod = getMinSupplierUnbondingPeriod(&defaultParams, paramDelta)

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedClaimWindowOpenOffestBlocks), defaultParams.ClaimWindowOpenOffsetBlocks)

	// Update the claim window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedClaimWindowOpenOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedClaimWindowOpenOffestBlocks), res.Params.ClaimWindowOpenOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ClaimWindowOpenOffsetBlocks")
}

func TestMsgUpdateParam_UpdateClaimWindowCloseOffsetBlocks(t *testing.T) {
	var expectedClaimWindowCloseOffestBlocks int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	defaultParams := sharedtypes.DefaultParams()

	// Get the delta between the previous claim window close offset blocks and its
	// new value to update the supplier unbonding period such as it is greater than
	// the cumulative proof window close blocks to pass UpdateParam validation.
	paramDelta := uint64(expectedClaimWindowCloseOffestBlocks) - defaultParams.ClaimWindowCloseOffsetBlocks
	defaultParams.SupplierUnbondingPeriod = getMinSupplierUnbondingPeriod(&defaultParams, paramDelta)

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedClaimWindowCloseOffestBlocks), defaultParams.ClaimWindowCloseOffsetBlocks)

	// Update the claim window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedClaimWindowCloseOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedClaimWindowCloseOffestBlocks), res.Params.ClaimWindowCloseOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ClaimWindowCloseOffsetBlocks")
}

func TestMsgUpdateParam_UpdateProofWindowOpenOffsetBlocks(t *testing.T) {
	var expectedProofWindowOpenOffestBlocks int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	defaultParams := sharedtypes.DefaultParams()

	// Get the delta between the previous proof window open offset blocks and its
	// new value to update the supplier unbonding period blocks such as it is greater
	// than the cumulative proof window close blocks to pass UpdateParam validation.
	paramDelta := uint64(expectedProofWindowOpenOffestBlocks) - defaultParams.ProofWindowOpenOffsetBlocks
	defaultParams.SupplierUnbondingPeriod = getMinSupplierUnbondingPeriod(&defaultParams, paramDelta)

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedProofWindowOpenOffestBlocks), defaultParams.ProofWindowOpenOffsetBlocks)

	// Update the proof window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedProofWindowOpenOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedProofWindowOpenOffestBlocks), res.Params.ProofWindowOpenOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ProofWindowOpenOffsetBlocks")
}

func TestMsgUpdateParam_UpdateProofWindowCloseOffsetBlocks(t *testing.T) {
	var expectedProofWindowCloseOffestBlocks int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	defaultParams := sharedtypes.DefaultParams()

	// Get the delta between the previous proof window close offset blocks and its
	// new value to update the supplier unbonding period blocks such as it is greater
	// than the cumulative proof window close blocks to pass UpdateParam validation.
	paramDelta := uint64(expectedProofWindowCloseOffestBlocks) - defaultParams.ProofWindowCloseOffsetBlocks
	defaultParams.SupplierUnbondingPeriod = getMinSupplierUnbondingPeriod(&defaultParams, paramDelta)

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedProofWindowCloseOffestBlocks), defaultParams.ProofWindowCloseOffsetBlocks)

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedProofWindowCloseOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedProofWindowCloseOffestBlocks), res.Params.ProofWindowCloseOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ProofWindowCloseOffsetBlocks")
}

func TestMsgUpdateParam_UpdateGracePeriodEndOffsetBlocks(t *testing.T) {
	var expectedGracePeriodEndOffestBlocks int64 = 2

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	defaultParams := sharedtypes.DefaultParams()

	// Update the claim window open offset blocks which has to be at least equal to
	// GracePeriodEndOffsetBlocks to pass UpdateParam validation.
	defaultParams.ClaimWindowOpenOffsetBlocks = uint64(expectedGracePeriodEndOffestBlocks)

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedGracePeriodEndOffestBlocks), defaultParams.GetGracePeriodEndOffsetBlocks())

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamGracePeriodEndOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedGracePeriodEndOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedGracePeriodEndOffestBlocks), res.Params.GetGracePeriodEndOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "GracePeriodEndOffsetBlocks")
}

func TestMsgUpdateParam_UpdateSupplierUnbondingPeriod(t *testing.T) {
	var expectedSupplierUnbondingPerid int64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	defaultParams := sharedtypes.DefaultParams()
	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedSupplierUnbondingPerid), defaultParams.GetSupplierUnbondingPeriod())

	// Update the supplier unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriod,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedSupplierUnbondingPerid},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedSupplierUnbondingPerid), res.Params.GetSupplierUnbondingPeriod())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "SupplierUnbondingPeriod")

	// Ensure that a supplier unbonding period that is less than the cumulative
	// proof window close blocks is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriod,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: 1},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedParamInvalid)
}

// getMinSupplierUnbondingPeriod returns the minimum supplier unbonding period
// in session number that is greater than the cumulative proof window close blocks.
func getMinSupplierUnbondingPeriod(params *sharedtypes.Params, delta uint64) uint64 {
	newProofWindowCloseBlobcks := types.GetCumulatedProofWindowCloseBlocks(params) + delta

	return (newProofWindowCloseBlobcks / params.NumBlocksPerSession) + 1
}
