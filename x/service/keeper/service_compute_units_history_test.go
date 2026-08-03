package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const testCuprServiceId = "svc1"

// TestServiceComputeUnitsPerRelayAtHeight_SetGet verifies the at-height lookup
// returns the most recent entry with effective_height <= queryHeight.
func TestServiceComputeUnitsPerRelayAtHeight_SetGet(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)

	require.NoError(t, k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 10, testCuprServiceId, 100))
	require.NoError(t, k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 20, testCuprServiceId, 200))

	tests := []struct {
		name        string
		queryHeight int64
		expectCupr  uint64
		expectFound bool
	}{
		// No service in the store, so a query before the first entry finds neither
		// history nor a fallback service and returns (0, false).
		{name: "before first entry, no service", queryHeight: 9, expectCupr: 0, expectFound: false},
		{name: "exactly at first entry", queryHeight: 10, expectCupr: 100, expectFound: true},
		{name: "between entries resolves to earlier", queryHeight: 19, expectCupr: 100, expectFound: true},
		{name: "exactly at second entry", queryHeight: 20, expectCupr: 200, expectFound: true},
		{name: "after last entry resolves to latest", queryHeight: 1000, expectCupr: 200, expectFound: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cupr, found := k.GetServiceComputeUnitsPerRelayAtHeight(ctx, testCuprServiceId, test.queryHeight)
			require.Equal(t, test.expectFound, found)
			require.Equal(t, test.expectCupr, cupr)
		})
	}
}

// TestServiceComputeUnitsPerRelayAtHeight_FallbackToCurrent verifies that a service
// with no history falls back to its current (live) cupr.
func TestServiceComputeUnitsPerRelayAtHeight_FallbackToCurrent(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)

	k.SetService(ctx, sharedtypes.Service{
		Id:                   testCuprServiceId,
		ComputeUnitsPerRelay: 42,
		OwnerAddress:         "pokt1owner",
	})

	// No history recorded: any height falls back to the current cupr.
	cupr, found := k.GetServiceComputeUnitsPerRelayAtHeight(ctx, testCuprServiceId, 5)
	require.True(t, found)
	require.Equal(t, uint64(42), cupr)

	// A recorded entry takes precedence over the fallback at/after its height.
	require.NoError(t, k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 100, testCuprServiceId, 7))
	cupr, found = k.GetServiceComputeUnitsPerRelayAtHeight(ctx, testCuprServiceId, 150)
	require.True(t, found)
	require.Equal(t, uint64(7), cupr)

	// Below the entry, still falls back to current.
	cupr, found = k.GetServiceComputeUnitsPerRelayAtHeight(ctx, testCuprServiceId, 50)
	require.True(t, found)
	require.Equal(t, uint64(42), cupr)
}

// TestSnapshotServiceComputeUnitsPerRelayChange verifies that a cupr change on a
// pre-existing service (empty history) seeds the previous value at height 1 and
// records the new value at the next session boundary, so an in-flight session
// resolves to the OLD cupr and a session starting after the boundary gets the NEW one.
func TestSnapshotServiceComputeUnitsPerRelayChange(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	// Compute the boundary the same way the keeper does, from live shared params.
	sharedParams := sharedtypes.DefaultParams()
	nextSessionStart := sharedtypes.GetSessionEndHeight(&sharedParams, 100) + 1

	// Pre-existing service at cupr=100 with no history.
	k.SetService(sdkCtx, sharedtypes.Service{
		Id:                   testCuprServiceId,
		ComputeUnitsPerRelay: 100,
		OwnerAddress:         "pokt1owner",
	})

	require.NoError(t, k.SnapshotServiceComputeUnitsPerRelayChange(sdkCtx, testCuprServiceId, 100, 200))

	// History: {1 -> 100 (seeded baseline), nextSessionStart -> 200}.
	history := k.GetServiceComputeUnitsPerRelayHistoryForService(sdkCtx, testCuprServiceId)
	require.Len(t, history, 2)

	// A session that started before the change resolves to the OLD cupr.
	cupr, found := k.GetServiceComputeUnitsPerRelayAtHeight(sdkCtx, testCuprServiceId, 100)
	require.True(t, found)
	require.Equal(t, uint64(100), cupr)

	// The block right before the boundary still resolves to OLD.
	cupr, found = k.GetServiceComputeUnitsPerRelayAtHeight(sdkCtx, testCuprServiceId, nextSessionStart-1)
	require.True(t, found)
	require.Equal(t, uint64(100), cupr)

	// A session starting at/after the boundary gets the NEW cupr.
	cupr, found = k.GetServiceComputeUnitsPerRelayAtHeight(sdkCtx, testCuprServiceId, nextSessionStart)
	require.True(t, found)
	require.Equal(t, uint64(200), cupr)
}

// TestSnapshotServiceComputeUnitsPerRelayChange_ExistingHistoryNotReseeded verifies
// that a second change does NOT re-seed the baseline at height 1.
func TestSnapshotServiceComputeUnitsPerRelayChange_ExistingHistoryNotReseeded(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	k.SetService(sdkCtx, sharedtypes.Service{
		Id:                   testCuprServiceId,
		ComputeUnitsPerRelay: 100,
		OwnerAddress:         "pokt1owner",
	})

	// First change seeds baseline + new value.
	require.NoError(t, k.SnapshotServiceComputeUnitsPerRelayChange(sdkCtx, testCuprServiceId, 100, 200))
	firstLen := len(k.GetServiceComputeUnitsPerRelayHistoryForService(sdkCtx, testCuprServiceId))
	require.Equal(t, 2, firstLen)

	// Second change (at a later block) only appends the new value, no re-seed at 1.
	laterCtx := sdkCtx.WithBlockHeight(500)
	require.NoError(t, k.SnapshotServiceComputeUnitsPerRelayChange(laterCtx, testCuprServiceId, 200, 300))
	history := k.GetServiceComputeUnitsPerRelayHistoryForService(laterCtx, testCuprServiceId)
	require.Len(t, history, 3)

	// Exactly one entry at height 1 (the original baseline).
	var atHeightOne int
	for _, h := range history {
		if h.EffectiveHeight == 1 {
			atHeightOne++
			require.Equal(t, uint64(100), h.ComputeUnitsPerRelay)
		}
	}
	require.Equal(t, 1, atHeightOne)
}

// TestSnapshotServiceComputeUnitsPerRelayCreate verifies a new service seeds its
// initial cupr at the start of the session in which it was created, so that its first
// (possibly partial) session is pinned rather than resolving via the live-cupr fallback.
func TestSnapshotServiceComputeUnitsPerRelayCreate(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	sharedParams := sharedtypes.DefaultParams()
	currentSessionStart := sharedtypes.GetSessionStartHeight(&sharedParams, 100)

	require.NoError(t, k.SnapshotServiceComputeUnitsPerRelayCreate(sdkCtx, testCuprServiceId, 555))

	history := k.GetServiceComputeUnitsPerRelayHistoryForService(sdkCtx, testCuprServiceId)
	require.Len(t, history, 1)
	require.Equal(t, currentSessionStart, history[0].EffectiveHeight)
	require.Equal(t, uint64(555), history[0].ComputeUnitsPerRelay)
	require.Equal(t, testCuprServiceId, history[0].ServiceId)

	// The created cupr must resolve for a lookup at the creation session start (the
	// first session), not fall back to the live value.
	cupr, found := k.GetServiceComputeUnitsPerRelayAtHeight(sdkCtx, testCuprServiceId, currentSessionStart)
	require.True(t, found)
	require.Equal(t, uint64(555), cupr)
}

// TestGetAllServiceComputeUnitsPerRelayHistory verifies cross-service enumeration.
func TestGetAllServiceComputeUnitsPerRelayHistory(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)

	require.NoError(t, k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 1, "svcA", 10))
	require.NoError(t, k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 5, "svcA", 20))
	require.NoError(t, k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 1, "svcB", 30))

	all := k.GetAllServiceComputeUnitsPerRelayHistory(ctx)
	require.Len(t, all, 3)

	// Per-service enumeration returns only that service's entries.
	require.Len(t, k.GetServiceComputeUnitsPerRelayHistoryForService(ctx, "svcA"), 2)
	require.Len(t, k.GetServiceComputeUnitsPerRelayHistoryForService(ctx, "svcB"), 1)
}
