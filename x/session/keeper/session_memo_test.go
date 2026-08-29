package keeper_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
)

// newMemoTestKeeper returns a session keeper, a context at block 100 in the
// given exec mode, and a request for a session that has suppliers at height 10.
func newMemoTestKeeper(t *testing.T, execMode sdk.ExecMode) (keeper.Keeper, sdk.Context, *types.QueryGetSessionRequest) {
	t.Helper()

	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(100).WithExecMode(execMode)
	req := &types.QueryGetSessionRequest{
		ApplicationAddress: keepertest.TestApp1Address,
		ServiceId:          keepertest.TestServiceId1,
		BlockHeight:        10,
	}
	return sessionKeeper, sdkCtx, req
}

// requireSessionsEqual asserts two sessions are equal on the wire. A memo hit
// is a proto round-trip, which canonicalizes empty repeated fields to nil
// exactly like a supplier read from the store does.
func requireSessionsEqual(t *testing.T, expected, actual *types.Session) {
	t.Helper()

	expectedBz, err := expected.Marshal()
	require.NoError(t, err)
	actualBz, err := actual.Marshal()
	require.NoError(t, err)
	require.Equal(t, expectedBz, actualBz)
}

func TestGetSession_Memo_OnlyPopulatedDuringFinalizeBlock(t *testing.T) {
	tests := []struct {
		desc            string
		execMode        sdk.ExecMode
		expectedEntries int
	}{
		{desc: "check", execMode: sdk.ExecModeCheck},
		{desc: "recheck", execMode: sdk.ExecModeReCheck},
		{desc: "simulate", execMode: sdk.ExecModeSimulate},
		{desc: "prepare_proposal", execMode: sdk.ExecModePrepareProposal},
		{desc: "process_proposal", execMode: sdk.ExecModeProcessProposal},
		{desc: "finalize", execMode: sdk.ExecModeFinalize, expectedEntries: 1},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			sessionKeeper, ctx, req := newMemoTestKeeper(t, test.execMode)

			_, err := sessionKeeper.GetSession(ctx, req)
			require.NoError(t, err)
			require.Equal(t, test.expectedEntries, keepertest.CountSessionMemoEntries(t, ctx))
		})
	}
}

func TestGetSession_Memo_HitEqualsFreshHydration(t *testing.T) {
	sessionKeeper, checkCtx, req := newMemoTestKeeper(t, sdk.ExecModeCheck)

	// Fresh hydration outside FinalizeBlock: the memo is never consulted.
	freshRes, err := sessionKeeper.GetSession(checkCtx, req)
	require.NoError(t, err)
	require.Equal(t, 0, keepertest.CountSessionMemoEntries(t, checkCtx))

	// First FinalizeBlock call populates the memo, second one is served from it.
	finalizeCtx := checkCtx.WithExecMode(sdk.ExecModeFinalize)
	missRes, err := sessionKeeper.GetSession(finalizeCtx, req)
	require.NoError(t, err)
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, finalizeCtx))

	hitRes, err := sessionKeeper.GetSession(finalizeCtx, req)
	require.NoError(t, err)
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, finalizeCtx))

	requireSessionsEqual(t, freshRes.GetSession(), missRes.GetSession())
	requireSessionsEqual(t, freshRes.GetSession(), hitRes.GetSession())

	// A hit is a fresh copy: mutating it MUST NOT leak into later hits.
	hitRes.GetSession().Suppliers = nil
	hitRes2, err := sessionKeeper.GetSession(finalizeCtx, req)
	require.NoError(t, err)
	requireSessionsEqual(t, freshRes.GetSession(), hitRes2.GetSession())

	// A different request height memoizes separately.
	otherReq := *req
	otherReq.BlockHeight = 11
	_, err = sessionKeeper.GetSession(finalizeCtx, &otherReq)
	require.NoError(t, err)
	require.Equal(t, 2, keepertest.CountSessionMemoEntries(t, finalizeCtx))
}

func TestGetSession_Memo_IsNotGasMetered(t *testing.T) {
	sessionKeeper, checkCtx, req := newMemoTestKeeper(t, sdk.ExecModeCheck)

	gasOf := func(ctx sdk.Context) uint64 {
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		_, err := sessionKeeper.GetSession(ctx, req)
		require.NoError(t, err)
		return ctx.GasMeter().GasConsumed()
	}

	// Simulation runs memo-off, so its estimate MUST bound the FinalizeBlock cost:
	// a miss costs exactly a fresh hydration, a hit strictly less.
	finalizeCtx := checkCtx.WithExecMode(sdk.ExecModeFinalize)
	freshGas := gasOf(checkCtx)
	missGas := gasOf(finalizeCtx)
	hitGas := gasOf(finalizeCtx)
	require.Equal(t, freshGas, missGas)
	require.Less(t, hitGas, missGas)
}

func TestGetSession_Memo_FailedTxWritesAreDiscarded(t *testing.T) {
	sessionKeeper, ctx, req := newMemoTestKeeper(t, sdk.ExecModeFinalize)

	// A tx executes on a branch of the block's multistore; a failed tx's branch
	// is never written back, so its memo entry vanishes with it.
	txCtx, writeCache := ctx.CacheContext()
	_, err := sessionKeeper.GetSession(txCtx, req)
	require.NoError(t, err)
	memoKey := types.SessionMemoKey(req.ApplicationAddress, req.ServiceId, req.BlockHeight)
	transientKey := keepertest.SessionTransientStoreKey(t, ctx)
	require.True(t, prefix.NewStore(txCtx.KVStore(transientKey), types.SessionMemoKeyPrefix).Has(memoKey))
	require.Equal(t, 0, keepertest.CountSessionMemoEntries(t, ctx))

	// A successful tx's branch is written back and later txs see the memo.
	writeCache()
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, ctx))
}

func TestGetSession_Memo_ClearedOnCommit(t *testing.T) {
	sessionKeeper, ctx, req := newMemoTestKeeper(t, sdk.ExecModeFinalize)

	_, err := sessionKeeper.GetSession(ctx, req)
	require.NoError(t, err)
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, ctx))

	// Commit ends the block: transient stores are wiped, nothing carries over.
	commitStore, ok := ctx.MultiStore().(storetypes.CommitMultiStore)
	require.True(t, ok)
	commitStore.Commit()
	require.Equal(t, 0, keepertest.CountSessionMemoEntries(t, ctx))
}

func TestGetSession_Memo_ErrorsAreNotMemoized(t *testing.T) {
	sessionKeeper, ctx, req := newMemoTestKeeper(t, sdk.ExecModeFinalize)

	// A service with no suppliers fails hydration; repeated calls keep failing
	// the same way rather than serving a cached error.
	req.ServiceId = keepertest.TestServiceId11
	for i := 0; i < 2; i++ {
		_, err := sessionKeeper.GetSession(ctx, req)
		require.ErrorContains(t, err, types.ErrSessionSuppliersNotFound.Error())
		require.Equal(t, 0, keepertest.CountSessionMemoEntries(t, ctx))
	}
}
