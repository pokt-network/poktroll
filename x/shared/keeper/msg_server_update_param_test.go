package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var testSharedParams = sharedtypes.Params{
	NumBlocksPerSession:                4,
	GracePeriodEndOffsetBlocks:         1,
	ClaimWindowOpenOffsetBlocks:        2,
	ClaimWindowCloseOffsetBlocks:       4,
	ProofWindowOpenOffsetBlocks:        0,
	ProofWindowCloseOffsetBlocks:       4,
	SupplierUnbondingPeriodSessions:    4,
	ApplicationUnbondingPeriodSessions: 4,
	ComputeUnitsToTokensMultiplier:     42,
}

func TestMsgUpdateParam_UpdateNumBlocksPerSession(t *testing.T) {
	var expectedNumBlocksPerSession int64 = 13

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedNumBlocksPerSession), testSharedParams.NumBlocksPerSession)

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
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, res.Params, "NumBlocksPerSession")
}

func TestMsgUpdateParam_UpdateClaimWindowOpenOffsetBlocks(t *testing.T) {
	var expectedClaimWindowOpenOffestBlocks int64 = 4

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		uint64(expectedClaimWindowOpenOffestBlocks),
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedClaimWindowOpenOffestBlocks), sharedParams.ClaimWindowOpenOffsetBlocks)

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
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, res.Params, "ClaimWindowOpenOffsetBlocks")
}

func TestMsgUpdateParam_UpdateClaimWindowCloseOffsetBlocks(t *testing.T) {
	var expectedClaimWindowCloseOffestBlocks int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		uint64(expectedClaimWindowCloseOffestBlocks),
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedClaimWindowCloseOffestBlocks), sharedParams.ClaimWindowCloseOffsetBlocks)

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
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, res.Params, "ClaimWindowCloseOffsetBlocks")
}

func TestMsgUpdateParam_UpdateProofWindowOpenOffsetBlocks(t *testing.T) {
	var expectedProofWindowOpenOffestBlocks int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		uint64(expectedProofWindowOpenOffestBlocks),
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedProofWindowOpenOffestBlocks), sharedParams.ProofWindowOpenOffsetBlocks)

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
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, res.Params, "ProofWindowOpenOffsetBlocks")
}

func TestMsgUpdateParam_UpdateProofWindowCloseOffsetBlocks(t *testing.T) {
	var expectedProofWindowCloseOffestBlocks int64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		uint64(expectedProofWindowCloseOffestBlocks),
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedProofWindowCloseOffestBlocks), sharedParams.ProofWindowCloseOffsetBlocks)

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
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, res.Params, "ProofWindowCloseOffsetBlocks")
}

func TestMsgUpdateParam_UpdateGracePeriodEndOffsetBlocks(t *testing.T) {
	var expectedGracePeriodEndOffestBlocks int64 = 2

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Update the claim window open offset blocks which has to be at least equal to
	// GracePeriodEndOffsetBlocks to pass UpdateParam validation.
	sharedParams.ClaimWindowOpenOffsetBlocks = uint64(expectedGracePeriodEndOffestBlocks)

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedGracePeriodEndOffestBlocks), sharedParams.GetGracePeriodEndOffsetBlocks())

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
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, res.Params, "GracePeriodEndOffsetBlocks")
}

func TestMsgUpdateParam_UpdateSupplierUnbondingPeriodSessions(t *testing.T) {
	var expectedSupplierUnbondingPerid int64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedSupplierUnbondingPerid), testSharedParams.GetSupplierUnbondingPeriodSessions())

	// Update the supplier unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedSupplierUnbondingPerid},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedSupplierUnbondingPerid), res.Params.GetSupplierUnbondingPeriodSessions())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, res.Params, "SupplierUnbondingPeriodSessions")

	// Ensure that a supplier unbonding period that is less than the cumulative
	// proof window close blocks is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: 1},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedParamInvalid)
}

func TestMsgUpdateParam_UpdateApplicationUnbondingPeriodSessions(t *testing.T) {
	var expectedApplicationUnbondingPerid int64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedApplicationUnbondingPerid), testSharedParams.GetApplicationUnbondingPeriodSessions())

	// Update the application unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamApplicationUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedApplicationUnbondingPerid},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedApplicationUnbondingPerid), res.Params.GetApplicationUnbondingPeriodSessions())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, res.Params, "ApplicationUnbondingPeriodSessions")

	// Ensure that a application unbonding period that is less than the cumulative
	// proof window close blocks is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamApplicationUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: 1},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedParamInvalid)
}

func TestMsgUpdateParam_ComputeUnitsToTokenMultiplier(t *testing.T) {
	var expectedComputeUnitsToTokenMultiplier int64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedComputeUnitsToTokenMultiplier), testSharedParams.GetComputeUnitsToTokensMultiplier())

	// Update the compute units to token multiplier param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitsToTokensMultiplier,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedComputeUnitsToTokenMultiplier},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedComputeUnitsToTokenMultiplier), res.Params.GetComputeUnitsToTokensMultiplier())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, res.Params, "ComputeUnitsToTokensMultiplier")

	// Ensure that compute units to token multiplier that is less than 1 is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitsToTokensMultiplier,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: 0},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.ErrorIs(t, err, sharedtypes.ErrSharedParamInvalid)
}

// getMinActorUnbondingPeriodSessions returns the actors unbonding period
// sessions such that it is greater than the cumulative proof window close blocks
// to pass UpdateParam validation.
func getMinActorUnbondingPeriodSessions(
	params *sharedtypes.Params,
	oldParamBlocksValue uint64,
	newParamBlocksValue uint64,
) uint64 {
	deltaBlocks := newParamBlocksValue - oldParamBlocksValue
	newProofWindowCloseBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(params) + deltaBlocks

	return (newProofWindowCloseBlocks / params.NumBlocksPerSession) + 1
}
