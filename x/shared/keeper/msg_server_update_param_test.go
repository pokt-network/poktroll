package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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
	ComputeUnitsToTokensMultiplier:     42,
}

func TestMsgUpdateParam_UpdateNumBlocksPerSession(t *testing.T) {
	var expectedNumBlocksPerSession uint64 = 13

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Set the parameters.
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedNumBlocksPerSession, sharedParams.NumBlocksPerSession)

	// Update the number of blocks per session
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedNumBlocksPerSession},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain num blocks per session is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedNumBlocksPerSession, params.GetNumBlocksPerSession())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.NumBlocksPerSession, params.GetNumBlocksPerSession())
	require.Equal(t, expectedNumBlocksPerSession, params.GetNumBlocksPerSession())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyNumBlocksPerSession))
}

func TestMsgUpdateParam_UpdateClaimWindowOpenOffsetBlocks(t *testing.T) {
	var expectedClaimWindowOpenOffsetBlocks uint64 = 4

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedClaimWindowOpenOffsetBlocks,
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedClaimWindowOpenOffsetBlocks, sharedParams.ClaimWindowOpenOffsetBlocks)

	// Update the claim window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedClaimWindowOpenOffsetBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain claim window open offset blocks is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedClaimWindowOpenOffsetBlocks, params.GetClaimWindowOpenOffsetBlocks())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.ClaimWindowOpenOffsetBlocks, params.GetClaimWindowOpenOffsetBlocks())
	require.Equal(t, expectedClaimWindowOpenOffsetBlocks, params.GetClaimWindowOpenOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyClaimWindowOpenOffsetBlocks))
}

func TestMsgUpdateParam_UpdateClaimWindowCloseOffsetBlocks(t *testing.T) {
	var expectedClaimWindowCloseOffsetBlocks uint64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedClaimWindowCloseOffsetBlocks,
	)

	// Update the SupplierUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.SupplierUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Update the ApplicationUnbondingPeriodSessions such that it is greater than the
	// cumulative proof window close blocks to pass UpdateParam validation.
	sharedParams.ApplicationUnbondingPeriodSessions = minUnbodningPeriodSessions

	// Set the parameters to their default values
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedClaimWindowCloseOffsetBlocks, sharedParams.ClaimWindowCloseOffsetBlocks)

	// Update the claim window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedClaimWindowCloseOffsetBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain claim window close offset blocks is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedClaimWindowCloseOffsetBlocks, params.GetClaimWindowCloseOffsetBlocks())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.ClaimWindowCloseOffsetBlocks, params.GetClaimWindowCloseOffsetBlocks())
	require.Equal(t, expectedClaimWindowCloseOffsetBlocks, params.GetClaimWindowCloseOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyClaimWindowCloseOffsetBlocks))
}

func TestMsgUpdateParam_UpdateProofWindowOpenOffsetBlocks(t *testing.T) {
	var expectedProofWindowOpenOffsettBlocks uint64 = 8

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Calculate the minimum unbonding period sessions required by the staking actors
	// to pass UpdateParam validation.
	minUnbodningPeriodSessions := getMinActorUnbondingPeriodSessions(
		&sharedParams,
		sharedParams.ClaimWindowOpenOffsetBlocks,
		expectedProofWindowOpenOffsettBlocks,
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
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofWindowOpenOffsettBlocks, sharedParams.ProofWindowOpenOffsetBlocks)

	// Update the proof window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedProofWindowOpenOffsettBlocks},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain proof window open offset blocks is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedProofWindowOpenOffsettBlocks, params.GetProofWindowOpenOffsetBlocks())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.ProofWindowOpenOffsetBlocks, params.GetProofWindowOpenOffsetBlocks())
	require.Equal(t, expectedProofWindowOpenOffsettBlocks, params.GetProofWindowOpenOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyProofWindowOpenOffsetBlocks))
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
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

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

	// Assert that the onchain proof window close offset blocks is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedProofWindowCloseOffestBlocks, params.GetProofWindowCloseOffsetBlocks())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.ProofWindowCloseOffsetBlocks, params.GetProofWindowCloseOffsetBlocks())
	require.Equal(t, expectedProofWindowCloseOffestBlocks, params.GetProofWindowCloseOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyProofWindowCloseOffsetBlocks))
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
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

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

	// Assert that the onchain grace period end offset block is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedGracePeriodEndOffestBlocks, params.GetGracePeriodEndOffsetBlocks())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.GracePeriodEndOffsetBlocks, params.GetGracePeriodEndOffsetBlocks())
	require.Equal(t, expectedGracePeriodEndOffestBlocks, params.GetGracePeriodEndOffsetBlocks())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyGracePeriodEndOffsetBlocks))
}

func TestMsgUpdateParam_UpdateSupplierUnbondingPeriodSessions(t *testing.T) {
	var expectedSupplierUnbondingPeriod uint64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Set the parameters.
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedSupplierUnbondingPeriod, sharedParams.GetSupplierUnbondingPeriodSessions())

	// Update the supplier unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamSupplierUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedSupplierUnbondingPeriod},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain supplier unbonding period is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedSupplierUnbondingPeriod, params.GetSupplierUnbondingPeriodSessions())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.SupplierUnbondingPeriodSessions, params.GetSupplierUnbondingPeriodSessions())
	require.Equal(t, expectedSupplierUnbondingPeriod, params.GetSupplierUnbondingPeriodSessions())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeySupplierUnbondingPeriodSessions))

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
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Set the parameters.
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedApplicationUnbondingPerid, sharedParams.GetApplicationUnbondingPeriodSessions())

	// Update the application unbonding period param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamApplicationUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedApplicationUnbondingPerid},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain application unbonding period is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedApplicationUnbondingPerid, params.GetApplicationUnbondingPeriodSessions())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.ApplicationUnbondingPeriodSessions, params.GetApplicationUnbondingPeriodSessions())
	require.Equal(t, expectedApplicationUnbondingPerid, params.GetApplicationUnbondingPeriodSessions())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyApplicationUnbondingPeriodSessions))

	// Ensure that a application unbonding period that is less than the cumulative
	// proof window close blocks is not allowed.
	invalidAppUnbondingPeriodSessions := 1
	updateParamMsg = &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamApplicationUnbondingPeriodSessions,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: uint64(invalidAppUnbondingPeriodSessions)},
	}
	_, err = msgSrv.UpdateParam(ctx, updateParamMsg)
	require.EqualError(t, err, status.Error(
		codes.InvalidArgument,
		sharedtypes.ErrSharedParamInvalid.Wrapf(
			"ApplicationUnbondingPeriodSessions (%v session) (%v blocks) must be greater than the cumulative ProofWindowCloseOffsetBlocks (%v)",
			invalidAppUnbondingPeriodSessions,
			sharedParams.NumBlocksPerSession,
			sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams),
		).Error(),
	).Error())
}

func TestMsgUpdateParam_ComputeUnitsToTokenMultiplier(t *testing.T) {
	var expectedComputeUnitsToTokenMultiplier uint64 = 5

	k, ctx := testkeeper.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)
	// Copy test params to avoid modifying them.
	sharedParams := testSharedParams

	// Set the parameters.
	require.NoError(t, k.SetInitialParams(ctx, sharedParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedComputeUnitsToTokenMultiplier, sharedParams.GetComputeUnitsToTokensMultiplier())

	// Update the compute units to token multiplier param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamComputeUnitsToTokensMultiplier,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedComputeUnitsToTokenMultiplier},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain compute units to token multiplier is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedComputeUnitsToTokenMultiplier, params.GetComputeUnitsToTokensMultiplier())

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateSharedParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, sharedParams.ComputeUnitsToTokensMultiplier, params.GetComputeUnitsToTokensMultiplier())
	require.Equal(t, expectedComputeUnitsToTokenMultiplier, params.GetComputeUnitsToTokensMultiplier())

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &sharedParams, &params, string(sharedtypes.KeyComputeUnitsToTokensMultiplier))

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
