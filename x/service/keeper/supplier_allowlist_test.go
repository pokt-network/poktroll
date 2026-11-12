package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/store/prefix"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
)

func TestSupplierAllowlistEntries(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	addr := sample.AccAddressBech32()

	require.False(t, k.HasSupplierAllowlistEntry(ctx, "svc1", addr))
	require.False(t, k.IsServicePermissioned(ctx, "svc1"))

	k.SetSupplierAllowlistEntry(ctx, "svc1", addr)
	require.True(t, k.HasSupplierAllowlistEntry(ctx, "svc1", addr))
	require.True(t, k.IsServicePermissioned(ctx, "svc1"))

	// Entries of "svc1" are invisible to "svc10" (and vice versa).
	require.False(t, k.IsServicePermissioned(ctx, "svc10"))
	require.False(t, k.HasSupplierAllowlistEntry(ctx, "svc10", addr))
	k.SetSupplierAllowlistEntry(ctx, "svc10", sample.AccAddressBech32())
	k.RemoveSupplierAllowlistEntry(ctx, "svc10", addr)
	require.True(t, k.HasSupplierAllowlistEntry(ctx, "svc1", addr))

	k.RemoveSupplierAllowlistEntry(ctx, "svc1", addr)
	require.False(t, k.HasSupplierAllowlistEntry(ctx, "svc1", addr))
	require.False(t, k.IsServicePermissioned(ctx, "svc1"))

	// Removing an absent entry is a no-op.
	k.RemoveSupplierAllowlistEntry(ctx, "svc1", addr)
	require.False(t, k.HasSupplierAllowlistEntry(ctx, "svc1", addr))
}

func TestIsSupplierAllowed(t *testing.T) {
	owner := sample.AccAddressBech32()
	operator := sample.AccAddressBech32()
	other := sample.AccAddressBech32()

	tests := []struct {
		desc     string
		listed   []string
		service  string
		expected bool
	}{
		{desc: "empty list allows everyone", listed: nil, service: "svc1", expected: true},
		{desc: "owner listed", listed: []string{owner}, service: "svc1", expected: true},
		{desc: "operator listed", listed: []string{operator}, service: "svc1", expected: true},
		{desc: "neither listed", listed: []string{other}, service: "svc1", expected: false},
		{desc: "unknown service allows everyone", listed: []string{other}, service: "unknown", expected: true},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			k, ctx := keepertest.ServiceKeeper(t)
			for _, addr := range test.listed {
				k.SetSupplierAllowlistEntry(ctx, "svc1", addr)
			}
			require.Equal(t, test.expected, k.IsSupplierAllowed(ctx, test.service, owner, operator))
		})
	}
}

// requireSweeps fails unless the pending sweeps are exactly expected, in order.
// A nil and an empty cursor both mean "not started".
func requireSweeps(t *testing.T, k keeper.Keeper, ctx context.Context, expected ...types.AllowlistSweep) {
	t.Helper()

	sweeps := k.GetPendingAllowlistSweeps(ctx)
	require.Len(t, sweeps, len(expected))
	for i, want := range expected {
		require.Equal(t, want.ServiceId, sweeps[i].ServiceId)
		if len(want.Cursor) == 0 {
			require.Empty(t, sweeps[i].Cursor, "sweep %q", want.ServiceId)
		} else {
			require.Equal(t, want.Cursor, sweeps[i].Cursor, "sweep %q", want.ServiceId)
		}
		require.Equal(t, want.Rerun, sweeps[i].Rerun, "sweep %q", want.ServiceId)
		require.Equal(t, want.DeferredFromHeight, sweeps[i].DeferredFromHeight, "sweep %q", want.ServiceId)
	}
}

func TestAllowlistSweepState(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	requireSweeps(t, k, ctx)

	// The initial sweep is deferred from the block height it was scheduled at.
	k.ScheduleInitialAllowlistSweep(sdkCtx.WithBlockHeight(10), "svcB")
	requireSweeps(t, k, ctx, types.AllowlistSweep{ServiceId: "svcB", DeferredFromHeight: 10})

	// A request with nothing pending starts a sweep now. Rows come back ordered
	// by service ID.
	k.RequestAllowlistSweep(ctx, "svcA")
	requireSweeps(t, k, ctx,
		types.AllowlistSweep{ServiceId: "svcA"},
		types.AllowlistSweep{ServiceId: "svcB", DeferredFromHeight: 10},
	)

	// A request changes nothing on a deferred sweep or on one that has not started,
	// whether its cursor is nil or an explicit empty slice.
	k.RequestAllowlistSweep(ctx, "svcB")
	k.RequestAllowlistSweep(ctx, "svcA")
	k.SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: "svcA", Cursor: []byte{}})
	k.RequestAllowlistSweep(ctx, "svcA")
	requireSweeps(t, k, ctx,
		types.AllowlistSweep{ServiceId: "svcA"},
		types.AllowlistSweep{ServiceId: "svcB", DeferredFromHeight: 10},
	)

	// A request on a started sweep sets rerun and keeps the cursor, once.
	k.SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: "svcA", Cursor: []byte{1}})
	k.RequestAllowlistSweep(ctx, "svcA")
	requireSweeps(t, k, ctx,
		types.AllowlistSweep{ServiceId: "svcA", Cursor: []byte{1}, Rerun: true},
		types.AllowlistSweep{ServiceId: "svcB", DeferredFromHeight: 10},
	)
	k.RequestAllowlistSweep(ctx, "svcA")
	requireSweeps(t, k, ctx,
		types.AllowlistSweep{ServiceId: "svcA", Cursor: []byte{1}, Rerun: true},
		types.AllowlistSweep{ServiceId: "svcB", DeferredFromHeight: 10},
	)

	// SetAllowlistSweep stores every field as given.
	full := types.AllowlistSweep{ServiceId: "svcB", Cursor: []byte("cursor"), Rerun: true, DeferredFromHeight: 5}
	k.SetAllowlistSweep(ctx, full)
	requireSweeps(t, k, ctx, types.AllowlistSweep{ServiceId: "svcA", Cursor: []byte{1}, Rerun: true}, full)

	// Scheduling an initial sweep overwrites whatever row is there.
	k.ScheduleInitialAllowlistSweep(sdkCtx.WithBlockHeight(20), "svcA")
	requireSweeps(t, k, ctx, types.AllowlistSweep{ServiceId: "svcA", DeferredFromHeight: 20}, full)

	k.ClearAllowlistSweep(ctx, "svcA")
	requireSweeps(t, k, ctx, full)

	// Clearing an absent sweep is a no-op.
	k.ClearAllowlistSweep(ctx, "svcA")
	k.ClearAllowlistSweep(ctx, "svcB")
	requireSweeps(t, k, ctx)
}

// A stored row that cannot be decoded must not panic the readers (the supplier
// EndBlocker is one of them): it is skipped when listing and replaced by a fresh
// immediate sweep when requested. The row's service ID is the store key.
func TestAllowlistSweepState_UndecodableRow(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	rootStore, ok := sdkCtx.MultiStore().(*rootmulti.Store)
	require.True(t, ok)
	rawKV, ok := rootStore.GetStoreByName(types.StoreKey).(storetypes.KVStore)
	require.True(t, ok)
	rawSweeps := prefix.NewStore(rawKV, types.KeyPrefix(types.SupplierAllowlistSweepKeyPrefix))

	// The service ID is not stored in the value: the key supplies it.
	noIdBz, err := (&types.AllowlistSweep{Cursor: []byte{9}}).Marshal()
	require.NoError(t, err)
	rawSweeps.Set([]byte("svcKey"), noIdBz)
	rawSweeps.Set([]byte("svcBad"), []byte{0xff, 0xff})

	requireSweeps(t, k, ctx, types.AllowlistSweep{ServiceId: "svcKey", Cursor: []byte{9}})

	k.RequestAllowlistSweep(ctx, "svcBad")
	requireSweeps(t, k, ctx,
		types.AllowlistSweep{ServiceId: "svcBad"},
		types.AllowlistSweep{ServiceId: "svcKey", Cursor: []byte{9}},
	)
}
