package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	GatewayUnbondingPeriodSessions:     4,
	// compute units to tokens multiplier in pPOKT (i.e. 1/compute_unit_cost_granularity)
	ComputeUnitsToTokensMultiplier: 42_000_000,
	// compute unit cost granularity is 1pPOKT (i.e. 1/1e6)
	ComputeUnitCostGranularity: 1_000_000,
}

func TestMsgUpdateParam_UpdateNumBlocksPerSession(t *testing.T) {
	var expectedNumBlocksPerSession uint64 = 13

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedNumBlocksPerSession, testSharedParams.NumBlocksPerSession)

	// Update the number of blocks per session
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedNumBlocksPerSession},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedNumBlocksPerSession, updatedParams.NumBlocksPerSession)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, &updatedParams, string(sharedtypes.KeyNumBlocksPerSession))
}

func TestMsgUpdateParam_UpdateClaimWindowOpenOffsetBlocks(t *testing.T) {
	var expectedClaimWindowOpenOffestBlocks uint64 = 4

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedClaimWindowOpenOffestBlocks,
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
	require.NotEqual(t, expectedClaimWindowOpenOffestBlocks, sharedParams.ClaimWindowOpenOffsetBlocks)

	// Update the claim window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedClaimWindowOpenOffestBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedClaimWindowOpenOffestBlocks, updatedParams.ClaimWindowOpenOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &updatedParams, string(sharedtypes.KeyClaimWindowOpenOffsetBlocks))
}

func TestMsgUpdateParam_UpdateClaimWindowCloseOffsetBlocks(t *testing.T) {
	var expectedClaimWindowCloseOffestBlocks uint64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedClaimWindowCloseOffestBlocks,
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
	require.NotEqual(t, expectedClaimWindowCloseOffestBlocks, sharedParams.ClaimWindowCloseOffsetBlocks)

	// Update the claim window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedClaimWindowCloseOffestBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedClaimWindowCloseOffestBlocks, updatedParams.ClaimWindowCloseOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &updatedParams, string(sharedtypes.KeyClaimWindowCloseOffsetBlocks))
}

func TestMsgUpdateParam_UpdateProofWindowOpenOffsetBlocks(t *testing.T) {
	var expectedProofWindowOpenOffestBlocks uint64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedProofWindowOpenOffestBlocks,
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the GatewayUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.GatewayUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofWindowOpenOffestBlocks, sharedParams.ProofWindowOpenOffsetBlocks)

	// Update the proof window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedProofWindowOpenOffestBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedProofWindowOpenOffestBlocks, updatedParams.ProofWindowOpenOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &updatedParams, string(sharedtypes.KeyProofWindowOpenOffsetBlocks))
}

func TestMsgUpdateParam_UpdateProofWindowCloseOffsetBlocks(t *testing.T) {
	var expectedProofWindowCloseOffestBlocks uint64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedProofWindowCloseOffestBlocks,
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
	require.NotEqual(t, expectedProofWindowCloseOffestBlocks, sharedParams.ProofWindowCloseOffsetBlocks)

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedProofWindowCloseOffestBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedProofWindowCloseOffestBlocks, updatedParams.ProofWindowCloseOffsetBlocks)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &updatedParams, string(sharedtypes.KeyProofWindowCloseOffsetBlocks))
}

func TestMsgUpdateParam_UpdateGracePeriodEndOffsetBlocks(t *testing.T) {
	var expectedGracePeriodEndOffestBlocks uint64 = 2

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Update the claim window open offset blocks which has to be at least equal to
	// GracePeriodEndOffsetBlocks to pass UpdateParam validation.
	sharedParams.ClaimWindowOpenOffsetBlocks = expectedGracePeriodEndOffestBlocks

	// Set the parameters to their default values
	require.NoError(t, k.SetParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedGracePeriodEndOffestBlocks, sharedParams.GetGracePeriodEndOffsetBlocks())

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamGracePeriodEndOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedGracePeriodEndOffestBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedGracePeriodEndOffestBlocks, updatedParams.GetGracePeriodEndOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &updatedParams, string(sharedtypes.KeyGracePeriodEndOffsetBlocks))
}

func TestMsgUpdateParam_UpdateSupplierUnbondingPeriodSessions(t *testing.T) {
	var expectedSupplierUnbondingPeriod uint64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedSupplierUnbondingPeriod, testSharedParams.GetSupplierUnbondingPeriodSessions())

	// Update the supplier unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedSupplierUnbondingPeriod},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedSupplierUnbondingPeriod, updatedParams.GetSupplierUnbondingPeriodSessions())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, &updatedParams, string(sharedtypes.KeySupplierUnbondingPeriodSessions))

	// Ensure that a supplier unbonding period that is less than the cumulative
	// proof window close blocks is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: 1},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.EqualError(t, err, status.Error(
		codes.InvalidArgument,
		sharedtypes.ErrSharedParamInvalid.Wrapf(
			"SupplierUnbondingPeriodSessions (%v session) (%v blocks) must be greater than the cumulative ProofWindowCloseOffsetBlocks (%v)",
			1, 4, 10,
		).Error(),
	).Error())
}

func TestMsgUpdateParam_UpdateApplicationUnbondingPeriodSessions(t *testing.T) {
	var expectedApplicationUnbondingPerid uint64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedApplicationUnbondingPerid, testSharedParams.GetApplicationUnbondingPeriodSessions())

	// Update the application unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamApplicationUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedApplicationUnbondingPerid},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedApplicationUnbondingPerid, updatedParams.GetApplicationUnbondingPeriodSessions())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, &updatedParams, string(sharedtypes.KeyApplicationUnbondingPeriodSessions))

	// Ensure that a application unbonding period that is less than the cumulative
	// proof window close blocks is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamApplicationUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: 1},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.EqualError(t, err, status.Error(
		codes.InvalidArgument,
		sharedtypes.ErrSharedParamInvalid.Wrapf(
			"ApplicationUnbondingPeriodSessions (%v session) (%v blocks) must be greater than the cumulative ProofWindowCloseOffsetBlocks (%v)",
			1, 4, 10,
		).Error(),
	).Error())
}

func TestMsgUpdateParam_ComputeUnitsToTokenMultiplier(t *testing.T) {
	var expectedComputeUnitsToTokenMultiplier uint64 = 5000000

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedComputeUnitsToTokenMultiplier, testSharedParams.GetComputeUnitsToTokensMultiplier())

	// Update the compute units to token multiplier param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitsToTokensMultiplier,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedComputeUnitsToTokenMultiplier},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedComputeUnitsToTokenMultiplier, updatedParams.GetComputeUnitsToTokensMultiplier())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, &updatedParams, string(sharedtypes.KeyComputeUnitsToTokensMultiplier))

	// Ensure that compute units to token multiplier that is less than 1 is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitsToTokensMultiplier,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: 0},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.EqualError(t, err, status.Error(
		codes.InvalidArgument,
		sharedtypes.ErrSharedParamInvalid.Wrapf(
			"invalid ComputeUnitsToTokensMultiplier: (%d)", 0,
		).Error(),
	).Error())
}

func TestMsgUpdateParam_ComputeUnitCostGranularity(t *testing.T) {
	var expectedComputeUnitCostGranularity uint64 = 1000

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters.
	require.NoError(t, k.SetParams(ctx, testSharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedComputeUnitCostGranularity, testSharedParams.GetComputeUnitCostGranularity())

	// Update the compute unit cost granularity param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitCostGranularity,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedComputeUnitCostGranularity},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.Equal(t, expectedComputeUnitCostGranularity, updatedParams.GetComputeUnitCostGranularity())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &testSharedParams, &updatedParams, string(sharedtypes.KeyComputeUnitCostGranularity))

	// Ensure that compute unit cost granularity that is less than 1 is not allowed.
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitCostGranularity,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: 0},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.EqualError(t, err, status.Error(
		codes.InvalidArgument,
		sharedtypes.ErrSharedParamInvalid.Wrapf(
			"invalid ComputeUnitCostGranularity: (%d)", 0,
		).Error(),
	).Error())
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
	newProofWindowCloseBlocks := uint64(sharedtypes.GetSessionEndToProofWindowCloseBlocks(params)) + deltaBlocks
	return (newProofWindowCloseBlocks / params.NumBlocksPerSession) + 1
}
