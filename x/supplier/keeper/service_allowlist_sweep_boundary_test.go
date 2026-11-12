package keeper_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// serviceConfigsOf returns the supplier's stored configs for serviceId.
func serviceConfigsOf(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, operatorAddr, serviceId string) []*sharedtypes.ServiceConfigUpdate {
	t.Helper()
	supplier, found := k.GetSupplier(ctx, operatorAddr)
	require.True(t, found)
	var configs []*sharedtypes.ServiceConfigUpdate
	for _, config := range supplier.ServiceConfigHistory {
		if config.Service.ServiceId == serviceId {
			configs = append(configs, config)
		}
	}
	return configs
}

// deactivationIndexSize returns the number of deactivation-height index entries.
func deactivationIndexSize(k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) int {
	store := prefix.NewStore(ctx.KVStore(k.StoreKey), suppliertypes.KeyPrefix(suppliertypes.ServiceConfigUpdateDeactivationHeightKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	defer iterator.Close()
	n := 0
	for ; iterator.Valid(); iterator.Next() {
		n++
	}
	return n
}

// sweepAt runs one EndBlocker sweep at the context's height and returns the
// number of visited configs and the not-allowed events it emitted.
func sweepAt(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) (int, []*suppliertypes.EventSupplierServiceNotAllowed) {
	t.Helper()
	sweepCtx := ctx.WithEventManager(cosmostypes.NewEventManager())
	n := k.EndBlockerServiceAllowlistSweep(sweepCtx)
	return n, notAllowedEvents(t, sweepCtx)
}

// A pass of 450 configs crosses a session boundary (200 per block, N=10): configs
// visited before the boundary deactivate at the next session start S, those visited
// after it at S+N, each exactly once. A removal mid-pass adds one rerun pass that
// deactivates nothing new.
func TestAllowlistSweep_CrossesSessionBoundary(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	params := k.SharedKeeper.GetParams(sdkCtx)
	operators := stakeOnService(t, k, sdkCtx, "svcB", 450)
	permissionAndMark(k, sdkCtx, "svcB")
	before := deactivationIndexSize(k, sdkCtx)

	S := sharedtypes.GetNextSessionStartHeight(&params, 19)
	N := int64(params.NumBlocksPerSession)
	require.Equal(t, int64(21), S)

	n, ev1 := sweepAt(t, k, sdkCtx.WithBlockHeight(S-2))
	require.Equal(t, 200, n)
	require.Len(t, ev1, 200)
	n, ev2 := sweepAt(t, k, sdkCtx.WithBlockHeight(S-1))
	require.Equal(t, 200, n)
	require.Len(t, ev2, 200)
	// A removal lands while the pass is running.
	k.ServiceKeeper.RequestAllowlistSweep(sdkCtx, "svcB")
	require.True(t, pendingSweep(t, k, sdkCtx, "svcB").Rerun)

	n, ev3 := sweepAt(t, k, sdkCtx.WithBlockHeight(S))
	require.Equal(t, 50, n)
	require.Len(t, ev3, 50)
	for _, e := range ev3 {
		require.Equal(t, S+N, e.GetEffectiveHeight(), "visited after the boundary: one session later")
	}
	require.Equal(t, []servicetypes.AllowlistSweep{{ServiceId: "svcB"}}, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))

	// Rerun pass: everything is already deactivating, so no new events, then cleared.
	total := 0
	for h := S + 1; len(k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx)) > 0; h++ {
		n, evr := sweepAt(t, k, sdkCtx.WithBlockHeight(h))
		require.Empty(t, evr)
		total += n
		require.Less(t, h, S+10)
	}
	require.Equal(t, 450, total)

	require.Equal(t, before+450, deactivationIndexSize(k, sdkCtx), "one index entry per config")
	atS, atSN := 0, 0
	for _, op := range operators {
		cfg := serviceConfigsOf(t, k, sdkCtx, op, "svcB")
		require.Len(t, cfg, 1)
		switch cfg[0].DeactivationHeight {
		case S:
			atS++
			require.False(t, cfg[0].IsActive(S))
		case S + N:
			atSN++
			require.True(t, cfg[0].IsActive(S))
			require.False(t, cfg[0].IsActive(S+N))
		default:
			t.Fatalf("unexpected deactivation height %d", cfg[0].DeactivationHeight)
		}
	}
	require.Equal(t, 400, atS)
	require.Equal(t, 50, atSN)
}

// Two pending services share the per-block budget across a session boundary:
// every config of both is visited exactly once and both sweeps finish.
func TestAllowlistSweep_TwoServicesAcrossBoundary(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	stakeOnService(t, k, sdkCtx, "svcB", 300)
	stakeOnService(t, k, sdkCtx, "svcC", 300)
	permissionAndMark(k, sdkCtx, "svcB")
	permissionAndMark(k, sdkCtx, "svcC")
	before := deactivationIndexSize(k, sdkCtx)

	events := map[string]int{}
	blocks := 0
	for h := int64(18); len(k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx)) > 0; h++ {
		_, ev := sweepAt(t, k, sdkCtx.WithBlockHeight(h))
		for _, e := range ev {
			events[e.GetOperatorAddress()+"/"+e.GetServiceId()]++
		}
		blocks++
		require.Less(t, blocks, 10)
	}
	require.Equal(t, 3, blocks)
	require.Len(t, events, 600)
	for key, count := range events {
		require.Equal(t, 1, count, key)
	}
	require.Equal(t, before+600, deactivationIndexSize(k, sdkCtx))
}
