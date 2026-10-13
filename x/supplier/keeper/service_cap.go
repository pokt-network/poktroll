package keeper

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// EnforceServiceCap schedules deactivation, at the next session start, of the
// supplier's services beyond what its stake covers, keeping services in
// service_priority order. It never unbonds a supplier: a stake below min_stake
// is left to the existing unbonding path. It never panics or returns an error.
//
// supplier must be the caller's live copy (in settlement, the settlement
// context's cached pointer), since callers may write it again afterwards.
func (k Keeper) EnforceServiceCap(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	reason suppliertypes.ServiceCapReason,
) (deactivatedServiceIds []string) {
	if supplier.IsUnbonding() || supplier.Stake == nil {
		return nil
	}

	allowed, limited := suppliertypes.AllowedServices(*supplier.Stake, k.GetParams(ctx))
	if !limited || allowed == 0 {
		return nil
	}

	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
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

	// Ranked holds the same pointers as serviceConfigHistory.
	ranked := suppliertypes.RankServiceConfigs(counted, supplier.ServicePriority)
	for _, config := range ranked[allowed:] {
		config.DeactivationHeight = nextSessionStartHeight
		deactivatedServiceIds = append(deactivatedServiceIds, config.Service.ServiceId)
	}

	supplierToIndex := *supplier
	supplierToIndex.ServiceConfigHistory = serviceConfigHistory
	k.SetAndIndexDehydratedSupplier(ctx, supplierToIndex)

	event := &suppliertypes.EventSupplierServicesCapped{
		OperatorAddress:       supplier.OperatorAddress,
		OwnerAddress:          supplier.OwnerAddress,
		Stake:                 supplier.Stake,
		AllowedServices:       allowed,
		DeactivatedServiceIds: deactivatedServiceIds,
		Reason:                reason,
		EffectiveHeight:       nextSessionStartHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		k.Logger().Error(fmt.Sprintf("failed to emit EventSupplierServicesCapped: %+v; %s", event, err))
	}

	return deactivatedServiceIds
}

// ServiceCapSweepBatchSize bounds the suppliers visited per block by the sweep.
// ponytail: fixed batch; tune from a mainnet-state measurement before the cap is enabled.
const ServiceCapSweepBatchSize = 200

// ScheduleServiceCapSweep starts (or restarts) a sweep over all suppliers.
// Call it whenever min_stake, min_stake_included_services or
// stake_per_additional_service changes, including from upgrade handlers.
func (k Keeper) ScheduleServiceCapSweep(ctx context.Context) {
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
