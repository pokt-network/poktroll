package keeper_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeOnService stakes n suppliers on serviceId, which must still be open.
func stakeOnService(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, serviceId string, n int) []string {
	t.Helper()
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	operators := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
		_, err := srv.StakeSupplier(ctx, multiServiceStakeMsg(t, k, ctx, ownerAddr, operatorAddr, capMinStake, serviceId))
		require.NoError(t, err)
		operators = append(operators, operatorAddr)
	}
	return operators
}

// permissionAndMark lists an unrelated address on serviceId and marks its sweep pending.
func permissionAndMark(k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, serviceId string) {
	k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, serviceId, sample.AccAddressBech32())
	k.ServiceKeeper.SetAllowlistSweep(ctx, servicetypes.AllowlistSweep{ServiceId: serviceId})
}

func notAllowedEvents(t *testing.T, ctx cosmostypes.Context) []*suppliertypes.EventSupplierServiceNotAllowed {
	t.Helper()
	return testutilevents.FilterEvents[*suppliertypes.EventSupplierServiceNotAllowed](t, ctx.EventManager().Events())
}

// sweepDeactivations runs one sweep and returns its deactivations by service ID.
func sweepDeactivations(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) map[string]int {
	t.Helper()
	sweepCtx := ctx.WithEventManager(cosmostypes.NewEventManager())
	k.EndBlockerServiceAllowlistSweep(sweepCtx)
	perService := map[string]int{}
	for _, event := range notAllowedEvents(t, sweepCtx) {
		perService[event.GetServiceId()]++
	}
	return perService
}

// lastSweptService returns the stored ID of the last service the sweep opened.
func lastSweptService(k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) string {
	return string(ctx.KVStore(k.StoreKey).Get(suppliertypes.AllowlistSweepLastServiceKey))
}

// pendingSweep returns the service's pending sweep, which must exist.
func pendingSweep(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, serviceId string) servicetypes.AllowlistSweep {
	t.Helper()
	for _, sweep := range k.ServiceKeeper.GetPendingAllowlistSweeps(ctx) {
		if sweep.ServiceId == serviceId {
			return sweep
		}
	}
	require.FailNow(t, "no pending sweep", serviceId)
	return servicetypes.AllowlistSweep{}
}

func TestAllowlistSweep_Batches(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	operators := stakeOnService(t, k, sdkCtx, "svcId", 450)
	permissionAndMark(k, sdkCtx, "svcId")

	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Len(t, notAllowedEvents(t, sdkCtx), 200)
	require.Len(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx), 1)

	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Len(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx), 1)

	require.Equal(t, 50, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
	require.Zero(t, k.EndBlockerServiceAllowlistSweep(sdkCtx))

	require.Len(t, notAllowedEvents(t, sdkCtx), 450)
	next := nextSessionStart(k, sdkCtx)
	for _, operatorAddr := range operators {
		require.Equal(t, map[string]int64{"svcId": next}, deactivationHeights(t, k, sdkCtx, operatorAddr))
	}
}

func TestAllowlistSweep_ExactBatchClears(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	stakeOnService(t, k, sdkCtx, "svcId", keeper.AllowlistSweepBatchSize)
	permissionAndMark(k, sdkCtx, "svcId")

	// Exactly one batch: cleared in the same call, not on the next.
	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
}

func TestAllowlistSweep_ServicesShareBudget(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	stakeOnService(t, k, sdkCtx, "a", 150)
	stakeOnService(t, k, sdkCtx, "b", 150)
	permissionAndMark(k, sdkCtx, "a")
	permissionAndMark(k, sdkCtx, "b")

	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	pending := k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx)
	require.Len(t, pending, 1)
	require.Equal(t, "b", pending[0].ServiceId)
	require.NotEmpty(t, pending[0].Cursor)

	perService := map[string]int{}
	for _, event := range notAllowedEvents(t, sdkCtx) {
		perService[event.GetServiceId()]++
	}
	require.Equal(t, map[string]int{"a": 150, "b": 50}, perService)

	require.Equal(t, 100, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
	require.Len(t, notAllowedEvents(t, sdkCtx), 300)
}

func TestAllowlistSweep_RoundRobin(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	for _, serviceId := range []string{"a", "b", "c"} {
		stakeOnService(t, k, sdkCtx, serviceId, 300)
		permissionAndMark(k, sdkCtx, serviceId)
	}

	require.Equal(t, map[string]int{"a": 200}, sweepDeactivations(t, k, sdkCtx))
	require.Equal(t, "a", lastSweptService(k, sdkCtx))
	require.Equal(t, map[string]int{"b": 200}, sweepDeactivations(t, k, sdkCtx))
	require.Equal(t, map[string]int{"c": 200}, sweepDeactivations(t, k, sdkCtx))

	// Wraps around: a finishes, then b finishes with the rest of the budget.
	require.Equal(t, map[string]int{"a": 100, "b": 100}, sweepDeactivations(t, k, sdkCtx))
	require.Equal(t, "b", lastSweptService(k, sdkCtx))
	pending := k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx)
	require.Len(t, pending, 1)
	require.Equal(t, "c", pending[0].ServiceId)
}

func TestAllowlistSweep_SpamDoesNotStarve(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	stakeOnService(t, k, sdkCtx, "a", 400)
	bOperators := stakeOnService(t, k, sdkCtx, "b", 150)
	permissionAndMark(k, sdkCtx, "a")
	permissionAndMark(k, sdkCtx, "b")

	// a's owner removes an entry before every block.
	for range 2 {
		k.ServiceKeeper.RequestAllowlistSweep(sdkCtx, "a")
		k.EndBlockerServiceAllowlistSweep(sdkCtx)
	}

	// b is done; a keeps its cursor and its requested rerun.
	pending := k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx)
	require.Len(t, pending, 1)
	require.Equal(t, "a", pending[0].ServiceId)
	require.NotEmpty(t, pending[0].Cursor)
	require.True(t, pending[0].Rerun)

	next := nextSessionStart(k, sdkCtx)
	for _, operatorAddr := range bOperators {
		require.Equal(t, map[string]int64{"b": next}, deactivationHeights(t, k, sdkCtx, operatorAddr))
	}
}

func TestAllowlistSweep_RerunRunsOneMorePass(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	stakeOnService(t, k, sdkCtx, "a", 180)
	stakeOnService(t, k, sdkCtx, "svcId", 50)
	permissionAndMark(k, sdkCtx, "a")
	permissionAndMark(k, sdkCtx, "svcId")

	// a takes 180 of the budget, so svcId's pass stops after 20 configs.
	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	k.ServiceKeeper.RequestAllowlistSweep(sdkCtx, "svcId")
	pending := k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx)
	require.Len(t, pending, 1)
	require.NotEmpty(t, pending[0].Cursor)
	require.True(t, pending[0].Rerun)

	// The pass finishes, then the rerun pass starts on the next call from the first config.
	require.Equal(t, 30, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Equal(t, []servicetypes.AllowlistSweep{{ServiceId: "svcId"}}, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
	require.Equal(t, 50, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
	require.Zero(t, k.EndBlockerServiceAllowlistSweep(sdkCtx))
}

func TestAllowlistSweep_DeferredInitial(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	operators := stakeOnService(t, k, sdkCtx, "svcId", 250)
	stakeOnService(t, k, sdkCtx, "svcOther", 400)
	permissionAndMark(k, sdkCtx, "svcOther")

	// Marked mid-session: the sweep is due at the next session start.
	sharedParams := k.SharedKeeper.GetParams(sdkCtx)
	markCtx := sdkCtx.WithBlockHeight(3)
	dueHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, 3)
	k.ServiceKeeper.SetSupplierAllowlistEntry(markCtx, "svcId", sample.AccAddressBech32())
	k.ServiceKeeper.ScheduleInitialAllowlistSweep(markCtx, "svcId")
	deferred := pendingSweep(t, k, sdkCtx, "svcId")
	require.Equal(t, int64(3), deferred.DeferredFromHeight)

	// Before it is due, svcId is skipped without spending budget.
	for _, height := range []int64{3, dueHeight - 1} {
		require.Equal(t, map[string]int{"svcOther": 200}, sweepDeactivations(t, k, sdkCtx.WithBlockHeight(height)))
		require.Equal(t, deferred, pendingSweep(t, k, sdkCtx, "svcId"))
	}
	require.Zero(t, k.EndBlockerServiceAllowlistSweep(sdkCtx.WithBlockHeight(dueHeight-1)))
	require.Equal(t, deferred, pendingSweep(t, k, sdkCtx, "svcId"))
	require.Equal(t, "svcOther", lastSweptService(k, sdkCtx))

	dueCtx := sdkCtx.WithBlockHeight(dueHeight)
	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(dueCtx))
	started := pendingSweep(t, k, sdkCtx, "svcId")
	require.Zero(t, started.DeferredFromHeight)
	require.NotEmpty(t, started.Cursor)
	require.Equal(t, 50, k.EndBlockerServiceAllowlistSweep(dueCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))

	deactivationHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, dueHeight)
	for _, operatorAddr := range operators {
		require.Equal(t, map[string]int64{"svcId": deactivationHeight}, deactivationHeights(t, k, sdkCtx, operatorAddr))
	}
}

func TestAllowlistSweep_BudgetEndsBetweenServices(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	stakeOnService(t, k, sdkCtx, "a", keeper.AllowlistSweepBatchSize)
	permissionAndMark(k, sdkCtx, "a")
	bSweep := servicetypes.AllowlistSweep{ServiceId: "b", Cursor: []byte("x")}
	k.ServiceKeeper.SetAllowlistSweep(sdkCtx, bSweep)

	require.Equal(t, keeper.AllowlistSweepBatchSize, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Equal(t, []servicetypes.AllowlistSweep{bSweep}, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
	require.Equal(t, "a", lastSweptService(k, sdkCtx))
}

func TestAllowlistSweep_Skips(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	operators := stakeOnService(t, k, sdkCtx, "svcId", 4)
	unbondingOp, deactivatingOp, listedOp, deletedOp := operators[0], operators[1], operators[2], operators[3]

	// Unbonding, with its config still counted.
	unbonding, found := k.GetDehydratedSupplier(sdkCtx, unbondingOp)
	require.True(t, found)
	unbonding.UnstakeSessionEndHeight = 10
	k.SetDehydratedSupplier(sdkCtx, unbonding)

	// Already deactivating at the next session start.
	deactivating, found := k.GetDehydratedSupplier(sdkCtx, deactivatingOp)
	require.True(t, found)
	require.True(t, k.DeactivateSupplierService(sdkCtx, &deactivating, "svcId"))

	// Listed by owner.
	listed, found := k.GetDehydratedSupplier(sdkCtx, listedOp)
	require.True(t, found)
	k.ServiceKeeper.SetSupplierAllowlistEntry(sdkCtx, "svcId", listed.OwnerAddress)

	// Supplier record deleted, its config left behind.
	store := sdkCtx.KVStore(k.StoreKey)
	prefix.NewStore(store, suppliertypes.KeyPrefix(suppliertypes.SupplierOperatorKeyPrefix)).
		Delete(suppliertypes.SupplierOperatorKey(deletedOp))

	// An undecodable config record.
	serviceConfigsPrefix := append(suppliertypes.KeyPrefix(suppliertypes.ServiceConfigUpdateKeyPrefix), suppliertypes.StringKey("svcId")...)
	prefix.NewStore(store, serviceConfigsPrefix).Set([]byte{0xff, 0xff}, []byte{0xff})

	k.ServiceKeeper.SetAllowlistSweep(sdkCtx, servicetypes.AllowlistSweep{ServiceId: "svcId"})
	sweepCtx := sdkCtx.WithEventManager(cosmostypes.NewEventManager())
	require.Equal(t, 5, k.EndBlockerServiceAllowlistSweep(sweepCtx))
	require.Empty(t, notAllowedEvents(t, sweepCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))

	require.Equal(t, int64(sharedtypes.NoDeactivationHeight), deactivationHeights(t, k, sdkCtx, unbondingOp)["svcId"])
	require.Equal(t, nextSessionStart(k, sdkCtx), deactivationHeights(t, k, sdkCtx, deactivatingOp)["svcId"])
	require.Equal(t, int64(sharedtypes.NoDeactivationHeight), deactivationHeights(t, k, sdkCtx, listedOp)["svcId"])
}

func TestAllowlistSweep_ListEmptiedNoop(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	operators := stakeOnService(t, k, sdkCtx, "svcId", 1)

	// Pending with an empty list, and pending with no staked configs.
	k.ServiceKeeper.SetAllowlistSweep(sdkCtx, servicetypes.AllowlistSweep{ServiceId: "svcId"})
	k.ServiceKeeper.SetAllowlistSweep(sdkCtx, servicetypes.AllowlistSweep{ServiceId: "svcId2"})

	require.Equal(t, 1, k.EndBlockerServiceAllowlistSweep(sdkCtx))
	require.Empty(t, notAllowedEvents(t, sdkCtx))
	require.Empty(t, k.ServiceKeeper.GetPendingAllowlistSweeps(sdkCtx))
	require.Equal(t, int64(sharedtypes.NoDeactivationHeight), deactivationHeights(t, k, sdkCtx, operators[0])["svcId"])
}
