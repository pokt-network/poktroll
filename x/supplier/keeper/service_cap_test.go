package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliermodule "github.com/pokt-network/poktroll/x/supplier/module"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeThenLower stakes 3 services in the given order with enough stake for 3,
// then lowers the stored stake to cover `covered` services (as a slash would).
func stakeThenLower(t *testing.T, covered int64, serviceIds ...string) (keepertest.SupplierModuleKeepers, cosmostypes.Context, sharedtypes.Supplier) {
	t.Helper()
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+int64(len(serviceIds)-1)*capPrice, serviceIds...)
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	supplier, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	supplier.Stake = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake + (covered-1)*capPrice)}
	k.SetDehydratedSupplier(ctx, supplier)
	return k, sdkCtx, supplier
}

func countedServiceIds(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, operatorAddr string) []string {
	t.Helper()
	sharedParams := k.SharedKeeper.GetParams(ctx)
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
	supplier, found := k.GetSupplier(ctx, operatorAddr)
	require.True(t, found)
	ids := make([]string, 0)
	for _, config := range supplier.ServiceConfigHistory {
		if suppliertypes.IsCountedServiceConfig(config, next) {
			ids = append(ids, config.Service.ServiceId)
		}
	}
	return ids
}

func TestEnforceServiceCap_KeepsOperatorOrderThroughTheStore(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")

	deactivated := k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH)
	require.Equal(t, []string{"base"}, deactivated)
	require.ElementsMatch(t, []string{"sol", "eth"}, countedServiceIds(t, k, ctx, supplier.OperatorAddress))

	sharedParams := k.SharedKeeper.GetParams(ctx)
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
	events := testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, ctx.EventManager().Events())
	require.Len(t, events, 1)
	require.Equal(t, "base", events[0].GetDeactivatedServiceId())
	require.Equal(t, supplier.Stake.String(), events[0].GetStake())
	require.Equal(t, uint64(2), events[0].GetAllowedServices())
	require.Equal(t, next, events[0].GetEffectiveHeight())
	require.Equal(t, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH, events[0].GetReason())
}

func TestEnforceServiceCap_FallbackOrderForEmptyPriority(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	// As for a supplier that has not staked with services since the priority existed.
	k.SetServicePriority(ctx, supplier.OperatorAddress, nil)

	deactivated := k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS)
	require.Equal(t, []string{"sol"}, deactivated) // service ID order: base, eth, sol
}

func TestEnforceServiceCap_IdempotentAndUnderCapIsNoop(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	require.Len(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH), 1)
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))

	k2, ctx2, underCap := stakeThenLower(t, 3, "a", "b", "c")
	require.Empty(t, k2.EnforceServiceCap(ctx2, &underCap, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, ctx2.EventManager().Events()))
}

func TestEnforceServiceCap_NeverUnbondsAndSkipsUnbonding(t *testing.T) {
	// Stake below min stake: capped like a stake at min stake, never unbonded here.
	k, ctx, supplier := stakeThenLower(t, 0, "sol", "eth")
	require.Equal(t, []string{"eth"}, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
	stored, _ := k.GetDehydratedSupplier(ctx, supplier.OperatorAddress)
	require.False(t, stored.IsUnbonding())

	// Unbonding supplier: skipped.
	k2, ctx2, unbonding := stakeThenLower(t, 1, "sol", "eth")
	unbonding.UnstakeSessionEndHeight = 10
	require.Empty(t, k2.EnforceServiceCap(ctx2, &unbonding, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
}

func TestEnforceServiceCap_ZeroServicesUntouched(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	})
	require.NoError(t, err)

	supplier, _ := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.Empty(t, k.EnforceServiceCap(sdkCtx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS))
	stored, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	require.False(t, stored.IsUnbonding())
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events()))
}

func TestEnforceServiceCap_DisabledIsNoop(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 1, "sol", "eth", "base")
	params := k.Keeper.GetParams(ctx)
	params.StakePerAdditionalService = nil
	require.NoError(t, k.SetParams(ctx, params))
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS))
}

// Raising min_stake caps a supplier left below the new minimum like one at the
// new minimum: it must not keep more services than a supplier that topped up.
func TestServiceCapSweep_MinStakeRaiseCapsBelowMinSuppliers(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)

	stake := func(amount int64, serviceIds ...string) string {
		ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
		_, err := srv.StakeSupplier(ctx, multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, amount, serviceIds...))
		require.NoError(t, err)
		return operatorAddr
	}
	belowNewMin := stake(capMinStake+capPrice, "a1", "a2")
	atNewMin := stake(2*capMinStake, "b1", "b2")

	params := k.Keeper.GetParams(sdkCtx)
	newMinStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2*capMinStake)
	params.MinStake = &newMinStake
	require.NoError(t, k.SetParams(sdkCtx, params))
	for k.IsServiceCapSweepPending(sdkCtx) {
		k.EndBlockerServiceCapSweep(sdkCtx)
	}

	require.Equal(t, []string{"a1"}, countedServiceIds(t, k, sdkCtx, belowNewMin))
	require.Equal(t, []string{"b1"}, countedServiceIds(t, k, sdkCtx, atNewMin))
}

// Any params write that enables or changes the cap schedules a sweep, including
// genesis and upgrade handlers that call SetParams directly.
func TestSetParams_SchedulesSweepOnlyWhenCapEnabledAndChanged(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	params := k.Keeper.GetParams(sdkCtx)

	// Cap disabled: no sweep, even when a cap param changes.
	params.MinStakeIncludedServices = 2
	require.NoError(t, k.SetParams(sdkCtx, params))
	require.False(t, k.IsServiceCapSweepPending(sdkCtx))

	// Enabling the cap schedules one.
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, capPrice)
	params.StakePerAdditionalService = &price
	require.NoError(t, k.SetParams(sdkCtx, params))
	require.True(t, k.IsServiceCapSweepPending(sdkCtx))
	for k.IsServiceCapSweepPending(sdkCtx) {
		k.EndBlockerServiceCapSweep(sdkCtx)
	}

	// Rewriting the same params schedules nothing.
	require.NoError(t, k.SetParams(sdkCtx, params))
	require.False(t, k.IsServiceCapSweepPending(sdkCtx))
}

// The service priority survives a genesis export and import.
func TestServicePriority_GenesisRoundTrip(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake, "sol", "eth"))
	require.NoError(t, err)

	exported := suppliermodule.ExportGenesis(sdkCtx, *k.Keeper)
	require.Equal(t, []suppliertypes.SupplierServicePriority{
		{OperatorAddress: operatorAddr, ServiceIds: []string{"sol", "eth"}},
	}, exported.ServicePriorityList)

	k2, ctx2 := keepertest.SupplierKeeper(t)
	suppliermodule.InitGenesis(ctx2, *k2.Keeper, *exported)
	require.Equal(t, []string{"sol", "eth"}, k2.GetServicePriority(ctx2, operatorAddr))
}

// A config capped before it ever activated is not reported as activated.
func TestEnforceServiceCap_PendingConfigNotReportedActivated(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+capPrice, "sol", "eth"))
	require.NoError(t, err)

	// A slash in the same session leaves stake for one service only.
	supplier, found := k.GetDehydratedSupplier(sdkCtx, operatorAddr)
	require.True(t, found)
	lowered := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, capMinStake)
	supplier.Stake = &lowered
	k.SetDehydratedSupplier(sdkCtx, supplier)
	require.Equal(t, []string{"eth"}, k.EnforceServiceCap(sdkCtx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))

	sharedParams := k.SharedKeeper.GetParams(sdkCtx)
	activationCtx := sdkCtx.
		WithBlockHeight(sharedtypes.GetNextSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())).
		WithEventManager(cosmostypes.NewEventManager())
	_, err = k.BeginBlockerActivateSupplierServices(activationCtx)
	require.NoError(t, err)

	activated := make([]string, 0)
	for _, event := range testutilevents.FilterEvents[*suppliertypes.EventSupplierServiceConfigActivated](t, activationCtx.EventManager().Events()) {
		activated = append(activated, event.GetServiceId())
	}
	require.Equal(t, []string{"sol"}, activated)
}

// EnforceServiceCap rewrites only the configs it reschedules; the resulting store
// state must equal a full re-index of the same supplier.
func TestEnforceServiceCap_IndexesMatchFullReindex(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 1, "sol", "eth", "base")
	require.Len(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH), 2)
	afterCap := dumpSupplierModuleStore(t, k, ctx)

	hydrated, found := k.GetSupplier(ctx, supplier.OperatorAddress)
	require.True(t, found)
	k.SetAndIndexDehydratedSupplier(ctx, hydrated)
	require.Equal(t, afterCap, dumpSupplierModuleStore(t, k, ctx))
}

// dumpSupplierModuleStore returns every key/value of the supplier module store.
func dumpSupplierModuleStore(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) map[string]string {
	t.Helper()
	store := ctx.KVStore(k.StoreKey)
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	dump := make(map[string]string)
	for ; iterator.Valid(); iterator.Next() {
		dump[string(iterator.Key())] = string(iterator.Value())
	}
	return dump
}

// BeginSupplierUnbonding rewrites only the changed configs; the result matches
// re-indexing the whole supplier.
func TestBeginSupplierUnbonding_IndexesMatchFullReindex(t *testing.T) {
	// Cap one of three services first, so the history mixes deactivated and
	// active configs.
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	require.Len(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH), 1)

	sharedParams := k.SharedKeeper.GetParams(ctx)
	k.BeginSupplierUnbonding(ctx, &supplier, &sharedParams)
	afterUnbonding := dumpSupplierModuleStore(t, k, ctx)
	require.Empty(t, countedServiceIds(t, k, ctx, supplier.OperatorAddress))

	hydrated, found := k.GetSupplier(ctx, supplier.OperatorAddress)
	require.True(t, found)
	require.True(t, hydrated.IsUnbonding())
	k.SetAndIndexDehydratedSupplier(ctx, hydrated)
	require.Equal(t, afterUnbonding, dumpSupplierModuleStore(t, k, ctx))
}
