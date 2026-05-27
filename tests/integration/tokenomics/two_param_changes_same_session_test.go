package integration_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestTwoSessionTimingParamChanges_SameSession_LastOneWins covers the C3
// scenario the audit flagged in shared/keeper/msg_server_update_param.go:
// two governance UpdateParam messages for the same session-timing param in
// the SAME in-flight session must coalesce to the LAST one. Both writes
// share an effective_height (the next session boundary), so the second
// SetParamsAtHeight overwrites the first — the live promotion at the
// boundary must reflect the FINAL governance intent, not the intermediate.
//
// The test exercises num_blocks_per_session as a representative
// session-timing param; the same code path handles the other five
// (grace + claim/proof window open/close offsets) per the extended
// deferral on this branch.
func TestTwoSessionTimingParamChanges_SameSession_LastOneWins(t *testing.T) {
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil,
		testkeeper.WithDefaultModuleBalances(),
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)

	const (
		oldN           int64 = 4
		intermediateN  int64 = 5 // first proposal
		finalN         int64 = 7 // second proposal (must win)
		boundaryHeight int64 = oldN + 1
		insideSession  int64 = 2 // mid-session height for the proposals
	)

	// Anchored grid at the genesis with N=oldN. Unbonding periods chosen so
	// ValidateBasic accepts BOTH proposals (intermediateN=5 and finalN=7).
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = uint64(oldN)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.SupplierUnbondingPeriodSessions = 16
	sharedParams.ApplicationUnbondingPeriodSessions = 16
	sharedParams.GatewayUnbondingPeriodSessions = 16
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

	concreteShared, ok := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected a concrete shared keeper")

	sharedMsgSrv := sharedkeeper.NewMsgServerImpl(*concreteShared)
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// --- First UpdateParam: propose N = intermediateN mid-session -------------
	sdkCtx = sdkCtx.WithBlockHeight(insideSession)
	_, err := sharedMsgSrv.UpdateParam(sdkCtx, &sharedtypes.MsgUpdateParam{
		Authority: govAuthority,
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: uint64(intermediateN)},
	})
	require.NoError(t, err)

	// Live params still oldN — deferred (Option B).
	require.Equal(t, uint64(oldN), keepers.SharedKeeper.GetParams(sdkCtx).NumBlocksPerSession,
		"live params must not change before the session boundary")

	// Sanity: the pending history entry at the boundary should reflect
	// intermediateN at this point. If the test setup ever drifts and these
	// no longer coincide, the second-write assertion below would still
	// catch the regression — but this intermediate check makes the failure
	// mode legible.
	pendingEntry, entryFound := concreteShared.GetParamsHistoryEntry(sdkCtx, boundaryHeight)
	require.True(t, entryFound, "first UpdateParam must record a history entry at the next session boundary")
	require.Equal(t, uint64(intermediateN), pendingEntry.NumBlocksPerSession,
		"history entry after the first UpdateParam must carry intermediateN")

	// --- Second UpdateParam: override with finalN, still mid-session ---------
	// The block height has not advanced. The deferred-promotion machinery
	// must compute the same effective_height (next session boundary) and
	// overwrite the first history entry at that key.
	_, err = sharedMsgSrv.UpdateParam(sdkCtx, &sharedtypes.MsgUpdateParam{
		Authority: govAuthority,
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: uint64(finalN)},
	})
	require.NoError(t, err)

	// Live params still oldN — both writes are deferred.
	require.Equal(t, uint64(oldN), keepers.SharedKeeper.GetParams(sdkCtx).NumBlocksPerSession,
		"second UpdateParam must not change live before the session boundary either")

	// The history entry at the boundary must now reflect finalN (the second
	// write). If the second SetParamsAtHeight failed to overwrite the first
	// at the same effective_height key, this assertion would still see
	// intermediateN and fail.
	overwrittenEntry, entryFound := concreteShared.GetParamsHistoryEntry(sdkCtx, boundaryHeight)
	require.True(t, entryFound, "history entry at the boundary must still exist after the second UpdateParam")
	require.Equal(t, uint64(finalN), overwrittenEntry.NumBlocksPerSession,
		"second UpdateParam must OVERWRITE the first at the same effective_height — last one wins")

	// --- Advance to the boundary and run the shared EndBlocker --------------
	sdkCtx = sdkCtx.WithBlockHeight(boundaryHeight)
	require.NoError(t, concreteShared.EndBlocker(sdkCtx))

	// Live params at the boundary must reflect finalN (NOT intermediateN). If
	// the EndBlocker promoted the older value, this is where the bug surfaces.
	promotedLive := keepers.SharedKeeper.GetParams(sdkCtx).NumBlocksPerSession
	require.Equal(t, uint64(finalN), promotedLive,
		"EndBlocker must promote the final governance intent (finalN=%d) — not the intermediate (intermediateN=%d)",
		finalN, intermediateN)

	// Sanity: GetParamsAtHeight at the boundary should also resolve to finalN —
	// the at-height resolver and live-promotion must agree on the value the
	// session boundary advanced into.
	resolvedAtBoundary := concreteShared.GetParamsAtHeight(sdkCtx, boundaryHeight)
	require.Equal(t, uint64(finalN), resolvedAtBoundary.NumBlocksPerSession,
		"GetParamsAtHeight at the boundary must also resolve to finalN — at-height and live must agree post-promotion")
}
