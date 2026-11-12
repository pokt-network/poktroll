package keeper

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// AllowlistSweepBatchSize bounds the service configs visited per block by the
// allowlist sweep, across all pending services.
const AllowlistSweepBatchSize = 200

// EndBlockerServiceAllowlistSweep deactivates, at the next session start, the
// services of staked suppliers no longer allowed by a service's supplier
// allowlist. It visits up to AllowlistSweepBatchSize service configs per call,
// across pending services round-robin: starting with the first service after
// the last one it opened, wrapping around, each continuing from its cursor. A
// deferred sweep is skipped, without spending budget, until the session start
// after its deferral height. When a service's configs are exhausted, its sweep
// is cleared, or restarted from the first config on its next turn if a rerun
// was requested. It never returns an error: an error from an EndBlocker halts
// the chain.
func (k Keeper) EndBlockerServiceAllowlistSweep(ctx context.Context) (numProcessed int) {
	sweeps := k.serviceKeeper.GetPendingAllowlistSweeps(ctx)
	if len(sweeps) == 0 {
		return 0
	}
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	// Pending sweeps are in store key order, i.e. by service ID bytes.
	lastServiceId := string(store.Get(suppliertypes.AllowlistSweepLastServiceKey))
	first := max(slices.IndexFunc(sweeps, func(sweep servicetypes.AllowlistSweep) bool {
		return sweep.ServiceId > lastServiceId
	}), 0)

	var nextSessionStartHeight int64
	lastOpenedServiceId := ""
	for offset := range sweeps {
		sweep := sweeps[(first+offset)%len(sweeps)]

		// Never open a service with no budget left: it would reset its cursor.
		remaining := AllowlistSweepBatchSize - numProcessed
		if remaining <= 0 {
			break
		}

		if sweep.DeferredFromHeight != 0 {
			deferredParams := k.sharedKeeper.GetParamsAtHeight(ctx, sweep.DeferredFromHeight)
			if sdkCtx.BlockHeight() < sharedtypes.GetNextSessionStartHeight(&deferredParams, sweep.DeferredFromHeight) {
				continue
			}
			// A due deferred sweep is a fresh pass.
			sweep.Cursor = nil
		}

		if nextSessionStartHeight == 0 {
			nextSessionStartHeight = k.nextSessionStartHeight(ctx)
		}
		lastOpenedServiceId = sweep.ServiceId

		// Collect the batch first: no writes while the service config iterator is open.
		var start []byte
		if len(sweep.Cursor) > 0 {
			start = append(bytes.Clone(sweep.Cursor), 0x00)
		}
		iterator := k.getServiceConfigUpdatesByServiceStore(ctx, sweep.ServiceId).Iterator(start, nil)
		batch := make([]sharedtypes.ServiceConfigUpdate, 0, remaining)
		visited := 0
		var lastKey []byte
		for ; iterator.Valid() && visited < remaining; iterator.Next() {
			visited++
			lastKey = bytes.Clone(iterator.Key())
			var config sharedtypes.ServiceConfigUpdate
			if err := k.cdc.Unmarshal(iterator.Value(), &config); err != nil {
				k.Logger().Error(fmt.Sprintf("allowlist sweep: skipping undecodable service config for service %q at key %x: %v", sweep.ServiceId, lastKey, err))
				continue
			}
			batch = append(batch, config)
		}
		exhausted := !iterator.Valid()
		iterator.Close()
		numProcessed += visited

		for i := range batch {
			k.enforceServiceAllowlist(ctx, &batch[i], sweep.ServiceId, nextSessionStartHeight)
		}

		switch {
		case !exhausted:
			k.serviceKeeper.SetAllowlistSweep(ctx, servicetypes.AllowlistSweep{ServiceId: sweep.ServiceId, Cursor: lastKey, Rerun: sweep.Rerun})
		case sweep.Rerun:
			// The rerun pass starts from the first config on the service's next turn.
			k.serviceKeeper.SetAllowlistSweep(ctx, servicetypes.AllowlistSweep{ServiceId: sweep.ServiceId})
		default:
			k.serviceKeeper.ClearAllowlistSweep(ctx, sweep.ServiceId)
		}
	}

	if lastOpenedServiceId != "" {
		store.Set(suppliertypes.AllowlistSweepLastServiceKey, []byte(lastOpenedServiceId))
	}

	return numProcessed
}

// enforceServiceAllowlist deactivates the service for the config's supplier if
// the config is still counted and the supplier is bonded and not allowed.
func (k Keeper) enforceServiceAllowlist(
	ctx context.Context,
	config *sharedtypes.ServiceConfigUpdate,
	serviceId string,
	nextSessionStartHeight int64,
) {
	if !suppliertypes.IsCountedServiceConfig(config, nextSessionStartHeight) {
		return
	}

	supplier, found := k.GetDehydratedSupplier(ctx, config.OperatorAddress)
	if !found {
		k.Logger().Error(fmt.Sprintf("allowlist sweep: skipping service %q config of missing supplier %q", serviceId, config.OperatorAddress))
		return
	}
	if supplier.IsUnbonding() {
		return
	}

	if !k.serviceKeeper.IsSupplierAllowed(ctx, serviceId, supplier.OwnerAddress, supplier.OperatorAddress) {
		k.DeactivateSupplierService(ctx, &supplier, serviceId)
	}
}
