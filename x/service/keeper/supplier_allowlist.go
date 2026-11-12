package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
)

func (k Keeper) getSupplierAllowlistStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierAllowlistKeyPrefix))
}

func (k Keeper) getAllowlistSweepStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierAllowlistSweepKeyPrefix))
}

func (k Keeper) getSupplierAllowlistCountStore(ctx context.Context) storetypes.KVStore {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierAllowlistCountKeyPrefix))
}

// UpdateSupplierAllowlistEntries removes, then adds, addresses on a service's
// supplier allowlist and returns the ones that really changed, in input order.
// Removing an absent entry or adding a present one is a no-op. The entry count
// is written once per call.
func (k Keeper) UpdateSupplierAllowlistEntries(
	ctx context.Context,
	serviceId string,
	add, remove []string,
) (added, removed []string) {
	store := k.getSupplierAllowlistStore(ctx)
	for _, address := range remove {
		key := types.SupplierAllowlistKey(serviceId, address)
		if store.Has(key) {
			store.Delete(key)
			removed = append(removed, address)
		}
	}
	for _, address := range add {
		key := types.SupplierAllowlistKey(serviceId, address)
		if !store.Has(key) {
			store.Set(key, []byte{})
			added = append(added, address)
		}
	}

	delta := int64(len(added)) - int64(len(removed))
	if delta == 0 {
		return added, removed
	}
	count := int64(k.getSupplierAllowlistCount(ctx, serviceId)) + delta
	if count <= 0 {
		k.getSupplierAllowlistCountStore(ctx).Delete([]byte(serviceId))
	} else {
		k.setSupplierAllowlistCount(ctx, serviceId, uint64(count))
	}
	return added, removed
}

// SetSupplierAllowlistEntry adds an address to a service's supplier allowlist.
// Adding a present entry is a no-op.
func (k Keeper) SetSupplierAllowlistEntry(ctx context.Context, serviceId, address string) {
	k.UpdateSupplierAllowlistEntries(ctx, serviceId, []string{address}, nil)
}

// RemoveSupplierAllowlistEntry removes an address from a service's supplier
// allowlist. Removing an absent entry is a no-op.
func (k Keeper) RemoveSupplierAllowlistEntry(ctx context.Context, serviceId, address string) {
	k.UpdateSupplierAllowlistEntries(ctx, serviceId, nil, []string{address})
}

// getSupplierAllowlistCount returns the number of allowlist entries of a service.
func (k Keeper) getSupplierAllowlistCount(ctx context.Context, serviceId string) uint64 {
	bz := k.getSupplierAllowlistCountStore(ctx).Get([]byte(serviceId))
	if len(bz) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) setSupplierAllowlistCount(ctx context.Context, serviceId string, count uint64) {
	k.getSupplierAllowlistCountStore(ctx).Set([]byte(serviceId), binary.BigEndian.AppendUint64(nil, count))
}

// HasSupplierAllowlistEntry reports whether an address is on a service's supplier allowlist.
func (k Keeper) HasSupplierAllowlistEntry(ctx context.Context, serviceId, address string) bool {
	return k.getSupplierAllowlistStore(ctx).Has(types.SupplierAllowlistKey(serviceId, address))
}

// GetAllSupplierAllowlistEntries returns every allowlist entry of every service in
// store key order. The service ID and address are read from the key, which a
// service ID cannot make ambiguous: it never contains "/".
func (k Keeper) GetAllSupplierAllowlistEntries(ctx context.Context) []types.SupplierAllowlistEntry {
	iterator := storetypes.KVStorePrefixIterator(k.getSupplierAllowlistStore(ctx), []byte{})
	defer iterator.Close()

	var entries []types.SupplierAllowlistEntry
	for ; iterator.Valid(); iterator.Next() {
		serviceId, address, _ := strings.Cut(string(iterator.Key()), "/")
		entries = append(entries, types.SupplierAllowlistEntry{ServiceId: serviceId, Address: address})
	}

	return entries
}

// IsServicePermissioned reports whether a service has at least one allowlist entry.
// It is a single point read on purpose: it runs on the stake and allowlist tx
// paths and must cost the same whatever the block wrote before.
func (k Keeper) IsServicePermissioned(ctx context.Context, serviceId string) bool {
	return k.getSupplierAllowlistCountStore(ctx).Has([]byte(serviceId))
}

// IsSupplierAllowed reports whether a supplier may serve a service: always true
// for a service with an empty allowlist (including an unknown one), otherwise
// true iff the supplier's owner or operator address is listed. Revenue share
// addresses never count.
func (k Keeper) IsSupplierAllowed(
	ctx context.Context,
	serviceId, ownerAddress, operatorAddress string,
) bool {
	if !k.IsServicePermissioned(ctx, serviceId) {
		return true
	}

	return k.HasSupplierAllowlistEntry(ctx, serviceId, ownerAddress) ||
		k.HasSupplierAllowlistEntry(ctx, serviceId, operatorAddress)
}

// getAllowlistSweep returns a service's pending sweep. A row that cannot be
// decoded is logged and reported as absent: this never panics on stored data.
func (k Keeper) getAllowlistSweep(ctx context.Context, serviceId string) (sweep types.AllowlistSweep, found bool) {
	bz := k.getAllowlistSweepStore(ctx).Get([]byte(serviceId))
	if bz == nil {
		return types.AllowlistSweep{}, false
	}

	if err := k.cdc.Unmarshal(bz, &sweep); err != nil {
		k.Logger().Error(fmt.Sprintf("ignoring undecodable allowlist sweep of service %q: %v", serviceId, err))
		return types.AllowlistSweep{}, false
	}
	sweep.ServiceId = serviceId

	return sweep, true
}

// ScheduleInitialAllowlistSweep schedules the sweep of a service that just became
// permissioned. It is deferred from the current block height, so the owner can
// build the list over several transactions before the sweep starts. It
// overwrites any row of the service.
func (k Keeper) ScheduleInitialAllowlistSweep(ctx context.Context, serviceId string) {
	k.SetAllowlistSweep(ctx, types.AllowlistSweep{
		ServiceId:          serviceId,
		DeferredFromHeight: sdk.UnwrapSDKContext(ctx).BlockHeight(),
	})
}

// RequestAllowlistSweep asks for a sweep after an allowlist entry was removed,
// without ever restarting a running one:
//   - none pending: start one now;
//   - pending but not started (deferred, or empty cursor): no change, the pass
//     reads the latest list;
//   - started: set Rerun, keeping the cursor, so exactly one more pass follows.
func (k Keeper) RequestAllowlistSweep(ctx context.Context, serviceId string) {
	sweep, found := k.getAllowlistSweep(ctx, serviceId)
	switch {
	case !found:
		k.SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: serviceId})
	case sweep.DeferredFromHeight != 0 || len(sweep.Cursor) == 0 || sweep.Rerun:
		// Not started, or a rerun is already requested: nothing to add.
	default:
		sweep.Rerun = true
		k.SetAllowlistSweep(ctx, sweep)
	}
}

// SetAllowlistSweep stores a service's pending sweep, replacing any row. The
// marshaled value is never nil, so an otherwise empty sweep stays pending.
func (k Keeper) SetAllowlistSweep(ctx context.Context, sweep types.AllowlistSweep) {
	k.getAllowlistSweepStore(ctx).Set([]byte(sweep.ServiceId), k.cdc.MustMarshal(&sweep))
}

// ClearAllowlistSweep removes a service's pending supplier sweep.
func (k Keeper) ClearAllowlistSweep(ctx context.Context, serviceId string) {
	k.getAllowlistSweepStore(ctx).Delete([]byte(serviceId))
}

// GetPendingAllowlistSweeps returns all pending sweeps in store key order. A row
// that cannot be decoded is logged and skipped: this never panics on stored data.
func (k Keeper) GetPendingAllowlistSweeps(ctx context.Context) []types.AllowlistSweep {
	iterator := storetypes.KVStorePrefixIterator(k.getAllowlistSweepStore(ctx), []byte{})
	defer iterator.Close()

	var sweeps []types.AllowlistSweep
	for ; iterator.Valid(); iterator.Next() {
		serviceId := string(iterator.Key())

		var sweep types.AllowlistSweep
		if err := k.cdc.Unmarshal(iterator.Value(), &sweep); err != nil {
			k.Logger().Error(fmt.Sprintf("skipping undecodable allowlist sweep of service %q: %v", serviceId, err))
			continue
		}
		sweep.ServiceId = serviceId

		sweeps = append(sweeps, sweep)
	}

	return sweeps
}
