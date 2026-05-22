package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	// Anchor the test at a realistic mid-session height so the next session boundary is
	// well-defined. With N=4 anchored at block 1, height 2 is in session [1,4] → the change
	// becomes effective at block 5 (#543 anchored grid).
	ctx = ctx.WithBlockHeight(2)
	const expectedEffectiveHeight int64 = 5

	// Set the parameters (anchor=1, the genesis grid).
	startParams := testSharedParams
	startParams.SessionGridAnchorHeight = 1
	startParams.SessionNumberAtAnchor = 1
	require.NoError(t, k.SetParams(ctx, startParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedNumBlocksPerSession, startParams.NumBlocksPerSession)

	// Update the number of blocks per session
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedNumBlocksPerSession},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// NARROW Option B (#543): a num_blocks_per_session change is DEFERRED — live params still
	// carry the OLD value until the next session boundary so in-flight sessions keep the old N.
	liveParams := k.GetParams(ctx)
	require.Equal(t, startParams.NumBlocksPerSession, liveParams.NumBlocksPerSession,
		"num_blocks_per_session must not change live before the session boundary")

	// The new value is recorded in history at the next session boundary, with the grid
	// anchored there.
	effectiveParams := k.GetParamsAtHeight(ctx, expectedEffectiveHeight)
	require.Equal(t, expectedNumBlocksPerSession, effectiveParams.NumBlocksPerSession)
	require.Equal(t, uint64(expectedEffectiveHeight), effectiveParams.SessionGridAnchorHeight)

	// The shared EndBlocker promotes the new epoch to live at the effective height.
	boundaryCtx := ctx.WithBlockHeight(expectedEffectiveHeight)
	require.NoError(t, k.EndBlocker(boundaryCtx))
	promotedParams := k.GetParams(boundaryCtx)
	require.Equal(t, expectedNumBlocksPerSession, promotedParams.NumBlocksPerSession)
	require.Equal(t, uint64(expectedEffectiveHeight), promotedParams.SessionGridAnchorHeight)

	// Ensure the other parameters are unchanged by the promotion.
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &startParams, &promotedParams,
		string(sharedtypes.KeyNumBlocksPerSession),
		"SessionGridAnchorHeight",
		"SessionNumberAtAnchor",
	)
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

	// Update the claim window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedClaimWindowOpenOffestBlocks},
	}

	// A claim window offset is a session-timing param: deferred to the next session
	// boundary (#543 Option B) so in-flight claims keep the window they were created under.
	requireSessionTimingParamDeferred(t, k, msgSrv, ctx, sharedParams, updateParamMsg,
		string(sharedtypes.KeyClaimWindowOpenOffsetBlocks),
		func(p sharedtypes.Params) uint64 { return p.ClaimWindowOpenOffsetBlocks },
		expectedClaimWindowOpenOffestBlocks)
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

	// Update the claim window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedClaimWindowCloseOffestBlocks},
	}

	// A claim window offset is a session-timing param: deferred to the next session
	// boundary (#543 Option B) so in-flight claims keep the window they were created under.
	requireSessionTimingParamDeferred(t, k, msgSrv, ctx, sharedParams, updateParamMsg,
		string(sharedtypes.KeyClaimWindowCloseOffsetBlocks),
		func(p sharedtypes.Params) uint64 { return p.ClaimWindowCloseOffsetBlocks },
		expectedClaimWindowCloseOffestBlocks)
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

	// Update the proof window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedProofWindowOpenOffestBlocks},
	}

	// A proof window offset is a session-timing param: deferred to the next session
	// boundary (#543 Option B) so in-flight claims keep the window they were created under.
	requireSessionTimingParamDeferred(t, k, msgSrv, ctx, sharedParams, updateParamMsg,
		string(sharedtypes.KeyProofWindowOpenOffsetBlocks),
		func(p sharedtypes.Params) uint64 { return p.ProofWindowOpenOffsetBlocks },
		expectedProofWindowOpenOffestBlocks)
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

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedProofWindowCloseOffestBlocks},
	}

	// A proof window offset is a session-timing param: deferred to the next session
	// boundary (#543 Option B) so in-flight claims keep the window they were created under.
	requireSessionTimingParamDeferred(t, k, msgSrv, ctx, sharedParams, updateParamMsg,
		string(sharedtypes.KeyProofWindowCloseOffsetBlocks),
		func(p sharedtypes.Params) uint64 { return p.ProofWindowCloseOffsetBlocks },
		expectedProofWindowCloseOffestBlocks)
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

	// Update the grace period end offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamGracePeriodEndOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedGracePeriodEndOffestBlocks},
	}

	// The grace period end offset is a session-timing param: deferred to the next session
	// boundary (#543 Option B) so in-flight sessions keep the grace period they began under.
	requireSessionTimingParamDeferred(t, k, msgSrv, ctx, sharedParams, updateParamMsg,
		string(sharedtypes.KeyGracePeriodEndOffsetBlocks),
		func(p sharedtypes.Params) uint64 { return p.GracePeriodEndOffsetBlocks },
		expectedGracePeriodEndOffestBlocks)
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

// requireSessionTimingParamDeferred asserts that a session-timing param update is
// DEFERRED to the next session boundary (#543 Option B): the live params are
// unchanged immediately after UpdateParam, the new value is recorded in history at
// the next session boundary, and the shared EndBlocker promotes it to live at that
// height. baseParams must use the testSharedParams grid (N=4); the helper anchors it
// at block 1 and runs the update at block 2, so the boundary is block 5.
func requireSessionTimingParamDeferred(
	t *testing.T,
	k keeper.Keeper,
	msgSrv sharedtypes.MsgServer,
	ctx sdk.Context,
	baseParams sharedtypes.Params,
	updateMsg *sharedtypes.MsgUpdateParam,
	paramKey string,
	getField func(sharedtypes.Params) uint64,
	expected uint64,
) {
	t.Helper()

	// Anchor the genesis grid at block 1 and run the update at a mid-session height.
	// With N=4 anchored at block 1, height 2 is in session [1,4] → the change becomes
	// effective at block 5.
	const queryHeight int64 = 2
	const effectiveHeight int64 = 5
	ctx = ctx.WithBlockHeight(queryHeight)

	startParams := baseParams
	startParams.SessionGridAnchorHeight = 1
	startParams.SessionNumberAtAnchor = 1
	require.NoError(t, k.SetParams(ctx, startParams))

	require.NotEqual(t, expected, getField(startParams),
		"test setup: new value must differ from the starting value")

	_, err := msgSrv.UpdateParam(ctx, updateMsg)
	require.NoError(t, err)

	// Deferred: live params must NOT change before the session boundary.
	require.Equal(t, getField(startParams), getField(k.GetParams(ctx)),
		"session-timing param must not change live before the session boundary")

	// The new value is recorded in history at the next session boundary.
	require.Equal(t, expected, getField(k.GetParamsAtHeight(ctx, effectiveHeight)),
		"new value must be recorded at the next session boundary")

	// The shared EndBlocker promotes the new epoch to live at the effective height.
	boundaryCtx := ctx.WithBlockHeight(effectiveHeight)
	require.NoError(t, k.EndBlocker(boundaryCtx))
	promoted := k.GetParams(boundaryCtx)
	require.Equal(t, expected, getField(promoted),
		"EndBlocker must promote the new value to live at the boundary")

	// Other params unchanged by the promotion (ignore derived grid metadata).
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &startParams, &promoted,
		paramKey, "SessionGridAnchorHeight", "SessionNumberAtAnchor")
}
