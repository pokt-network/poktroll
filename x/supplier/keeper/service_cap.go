package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// EnforceServiceCap schedules deactivation, at the next session start, of the
// supplier's services beyond what its stake covers, keeping services in the
// supplier's service priority order. It never unbonds a supplier: a stake below
// min_stake is left to the existing unbonding path. It never panics or returns
// an error.
//
// It writes only the rescheduled service configs; the supplier record itself is
// unchanged. A caller holding a hydrated copy of this supplier must not re-index
// it with SetAndIndexDehydratedSupplier afterwards, which would undo the
// deactivation; writing it back with SetDehydratedSupplier is safe.
func (k Keeper) EnforceServiceCap(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	reason suppliertypes.ServiceCapReason,
) (deactivatedServiceIds []string) {
	if supplier.IsUnbonding() || supplier.Stake == nil {
		return nil
	}

	allowed, limited := suppliertypes.AllowedServices(*supplier.Stake, k.GetParams(ctx))
	if !limited {
		return nil
	}

	// The params effective at this height describe the current session: at the
	// block where a session length change takes effect, the live params still
	// hold the previous grid until the shared EndBlocker promotes the new one.
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := k.sharedKeeper.GetParamsAtHeight(ctx, sdkCtx.BlockHeight())
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())

	serviceConfigHistory := k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress, "")
	counted := make([]*sharedtypes.ServiceConfigUpdate, 0, len(serviceConfigHistory))
	for _, config := range serviceConfigHistory {
		if suppliertypes.IsCountedServiceConfig(config, nextSessionStartHeight) {
			counted = append(counted, config)
		}
	}
	if uint64(len(counted)) <= allowed {
		return nil
	}

	ranked := suppliertypes.RankServiceConfigs(counted, k.GetServicePriority(ctx, supplier.OperatorAddress))
	stake := supplier.Stake.String()
	for _, config := range ranked[allowed:] {
		k.rescheduleServiceConfigDeactivation(ctx, config, nextSessionStartHeight)
		deactivatedServiceIds = append(deactivatedServiceIds, config.Service.ServiceId)

		event := &suppliertypes.EventSupplierServicesCapped{
			OperatorAddress:      supplier.OperatorAddress,
			OwnerAddress:         supplier.OwnerAddress,
			Stake:                stake,
			AllowedServices:      allowed,
			DeactivatedServiceId: config.Service.ServiceId,
			Reason:               reason,
			EffectiveHeight:      nextSessionStartHeight,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
			k.Logger().Error(fmt.Sprintf("failed to emit EventSupplierServicesCapped: %+v; %s", event, err))
		}
	}

	return deactivatedServiceIds
}

// rescheduleServiceConfigDeactivation sets one stored service config's deactivation
// height, rewriting only its primary record and its deactivation-height index
// entry. The other indexes do not depend on the deactivation height.
func (k Keeper) rescheduleServiceConfigDeactivation(
	ctx context.Context,
	config *sharedtypes.ServiceConfigUpdate,
	deactivationHeight int64,
) {
	deactivationHeightStore := k.getServiceConfigUpdateDeactivationHeightStore(ctx)
	if config.DeactivationHeight != sharedtypes.NoDeactivationHeight {
		deactivationHeightStore.Delete(suppliertypes.ServiceConfigUpdateDeactivationHeightKey(*config))
	}

	config.DeactivationHeight = deactivationHeight
	primaryKey := suppliertypes.ServiceConfigUpdateKey(*config)
	k.getServiceConfigUpdatesStore(ctx).Set(primaryKey, k.cdc.MustMarshal(config))
	if deactivationHeight != sharedtypes.NoDeactivationHeight {
		deactivationHeightStore.Set(suppliertypes.ServiceConfigUpdateDeactivationHeightKey(*config), primaryKey)
	}
}

// SetServicePriority stores the service IDs of the supplier's latest stake
// message that listed services, in message order.
func (k Keeper) SetServicePriority(ctx context.Context, operatorAddress string, serviceIds []string) {
	priority := suppliertypes.SupplierServicePriority{OperatorAddress: operatorAddress, ServiceIds: serviceIds}
	k.getServicePriorityStore(ctx).Set(suppliertypes.SupplierOperatorKey(operatorAddress), k.cdc.MustMarshal(&priority))
}

// GetServicePriority returns the supplier's service priority, or nil if it has
// none (suppliers that have not staked with services since the priority existed).
func (k Keeper) GetServicePriority(ctx context.Context, operatorAddress string) []string {
	priorityBz := k.getServicePriorityStore(ctx).Get(suppliertypes.SupplierOperatorKey(operatorAddress))
	if priorityBz == nil {
		return nil
	}
	var priority suppliertypes.SupplierServicePriority
	k.cdc.MustUnmarshal(priorityBz, &priority)
	return priority.ServiceIds
}

// GetAllServicePriorities returns every stored service priority, in store key order.
func (k Keeper) GetAllServicePriorities(ctx context.Context) []suppliertypes.SupplierServicePriority {
	iterator := k.getServicePriorityStore(ctx).Iterator(nil, nil)
	defer iterator.Close()

	priorities := make([]suppliertypes.SupplierServicePriority, 0)
	for ; iterator.Valid(); iterator.Next() {
		var priority suppliertypes.SupplierServicePriority
		k.cdc.MustUnmarshal(iterator.Value(), &priority)
		priorities = append(priorities, priority)
	}
	return priorities
}

func (k Keeper) removeServicePriority(ctx context.Context, operatorAddress string) {
	k.getServicePriorityStore(ctx).Delete(suppliertypes.SupplierOperatorKey(operatorAddress))
}

func (k Keeper) getServicePriorityStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, suppliertypes.KeyPrefix(suppliertypes.ServicePriorityKeyPrefix))
}

// ServiceCapSweepBatchSize bounds the suppliers visited per block by the sweep.
// At 500, a sweep over ~4,600 suppliers takes about 10 blocks.
// ponytail: fixed batch; re-measure against mainnet state before the cap is enabled.
const ServiceCapSweepBatchSize = 500

// scheduleServiceCapSweep starts (or restarts) a sweep over all suppliers.
// setParams calls it when the cap is enabled and its params change, so every
// params write (genesis, upgrade handlers, governance) schedules it as needed.
func (k Keeper) scheduleServiceCapSweep(ctx context.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(suppliertypes.ServiceCapSweepKey, []byte{})
}

// IsServiceCapSweepPending reports whether a sweep is in progress.
func (k Keeper) IsServiceCapSweepPending(ctx context.Context) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return store.Has(suppliertypes.ServiceCapSweepKey)
}

// EndBlockerServiceCapSweep enforces the service cap on up to
// ServiceCapSweepBatchSize suppliers, in store key order, continuing from the
// cursor. It clears the sweep when the supplier store is exhausted or the cap is
// disabled. It never returns an error: an error from an EndBlocker halts the chain.
func (k Keeper) EndBlockerServiceCapSweep(ctx context.Context) (numProcessed int) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	cursor := store.Get(suppliertypes.ServiceCapSweepKey)
	if cursor == nil {
		return 0
	}

	if !k.GetParams(ctx).ServiceCapEnabled() {
		store.Delete(suppliertypes.ServiceCapSweepKey)
		return 0
	}

	// Collect the batch first: no writes while the supplier store iterator is open.
	supplierStore := prefix.NewStore(store, suppliertypes.KeyPrefix(suppliertypes.SupplierOperatorKeyPrefix))
	var start []byte
	if len(cursor) > 0 {
		start = append(bytes.Clone(cursor), 0x00)
	}
	iterator := supplierStore.Iterator(start, nil)
	batch := make([]sharedtypes.Supplier, 0, ServiceCapSweepBatchSize)
	var lastKey []byte
	for ; iterator.Valid() && len(batch) < ServiceCapSweepBatchSize; iterator.Next() {
		lastKey = bytes.Clone(iterator.Key())
		var supplier sharedtypes.Supplier
		if err := k.cdc.Unmarshal(iterator.Value(), &supplier); err != nil {
			k.Logger().Error(fmt.Sprintf("service cap sweep: skipping undecodable supplier at key %x: %v", lastKey, err))
			continue
		}
		batch = append(batch, supplier)
	}
	exhausted := !iterator.Valid()
	iterator.Close()

	for i := range batch {
		k.EnforceServiceCap(ctx, &batch[i], suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS)
	}

	if exhausted {
		store.Delete(suppliertypes.ServiceCapSweepKey)
	} else {
		store.Set(suppliertypes.ServiceCapSweepKey, lastKey)
	}

	return len(batch)
}

// serviceCapParamsChanged reports whether any param that sets the cap changed.
func serviceCapParamsChanged(before, after suppliertypes.Params) bool {
	return !coinsEqual(before.MinStake, after.MinStake) ||
		before.MinStakeIncludedServices != after.MinStakeIncludedServices ||
		!coinsEqual(before.StakePerAdditionalService, after.StakePerAdditionalService)
}

func coinsEqual(a, b *cosmostypes.Coin) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
