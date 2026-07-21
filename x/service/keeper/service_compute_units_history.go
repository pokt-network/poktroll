package keeper

import (
	"context"
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SetServiceComputeUnitsPerRelayAtHeight stores a snapshot of a service's
// compute_units_per_relay (cupr) with the height at which it became effective,
// for historical (session-start) lookups.
func (k Keeper) SetServiceComputeUnitsPerRelayAtHeight(
	ctx context.Context,
	effectiveHeight int64,
	serviceId string,
	computeUnitsPerRelay uint64,
) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	update := types.ServiceComputeUnitsPerRelayUpdate{
		EffectiveHeight:      effectiveHeight,
		ServiceId:            serviceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
	}

	bz, err := k.cdc.Marshal(&update)
	if err != nil {
		return err
	}

	key := types.ServiceComputeUnitsPerRelayHistoryKey(serviceId, effectiveHeight)
	store.Set(key, bz)

	return nil
}

// GetServiceComputeUnitsPerRelayAtHeight returns the compute_units_per_relay (cupr)
// that was effective at the given height for a service. It finds the most recent
// history entry with effective_height <= queryHeight.
//
// If no history entry exists at or before queryHeight it falls back to the service's
// current cupr. Unlike relay mining difficulty — whose "current" store could diverge
// across nodes, forcing a param-derived base fallback — the service store is plain
// consensus state, so the current cupr is deterministic and safe to return.
//
// Returns (0, false) only when the service itself does not exist.
func (k Keeper) GetServiceComputeUnitsPerRelayAtHeight(
	ctx context.Context,
	serviceId string,
	queryHeight int64,
) (uint64, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	serviceHistoryPrefix := types.ServiceComputeUnitsPerRelayHistoryKeyPrefixForService(serviceId)
	historyStore := prefix.NewStore(store, serviceHistoryPrefix)

	// Exclusive upper bound for reverse iteration: the most recent entry with
	// effective_height <= queryHeight.
	endKey := make([]byte, 8)
	binary.BigEndian.PutUint64(endKey, uint64(queryHeight+1))

	iterator := historyStore.ReverseIterator(nil, endKey)
	defer iterator.Close()

	if iterator.Valid() {
		var update types.ServiceComputeUnitsPerRelayUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &update)
		return update.ComputeUnitsPerRelay, true
	}

	// Fallback to the current cupr (deterministic consensus state). Reached only for
	// a service with no recorded history at or before queryHeight — i.e. a service
	// that predates this history (pre-upgrade) and has never changed cupr, or a query
	// height before the service's first recorded entry. The rollout freeze (no cupr
	// changes across the upgrade) guarantees the live cupr equals the historical cupr
	// for every claimable session in that window.
	service, found := k.GetService(ctx, serviceId)
	if !found {
		return 0, false
	}
	return service.ComputeUnitsPerRelay, true
}

// SnapshotServiceComputeUnitsPerRelayCreate records the initial cupr of a newly
// created service, effective from the start of the session in which it was created.
//
// Seeding at the current session start (not the next boundary) pins the service's first
// session: a supplier may serve relays and a claim may be created for the session that is
// already in flight when the service is created, and that claim's session-start lookup
// must resolve to this initial cupr instead of falling back to the mutable live value.
func (k Keeper) SnapshotServiceComputeUnitsPerRelayCreate(
	ctx context.Context,
	serviceId string,
	computeUnitsPerRelay uint64,
) error {
	return k.SetServiceComputeUnitsPerRelayAtHeight(
		ctx,
		k.currentSessionStartHeight(ctx),
		serviceId,
		computeUnitsPerRelay,
	)
}

// SnapshotServiceComputeUnitsPerRelayChange records a cupr change for a service so
// historical (session-start) lookups return the correct value.
//
// The new cupr becomes effective at the NEXT session boundary — mirroring relay
// mining difficulty — so an in-flight session always resolves to the cupr that was
// live at its start. This is what eliminates the mid-session cupr flip that forfeited
// claims with ErrProofComputeUnitsMismatch.
//
// When the service has no cupr history yet (a pre-upgrade service changing cupr for
// the first time), prevCupr is seeded at height 1 first so that every session that
// started before this change resolves to the old value. Height 1 (not the current
// height, as relay mining difficulty uses) is required: a cupr change lands at an
// arbitrary height mid-session, and the in-flight session started strictly before the
// current height, so the baseline must sort before any claimable session start.
func (k Keeper) SnapshotServiceComputeUnitsPerRelayChange(
	ctx context.Context,
	serviceId string,
	prevCupr uint64,
	newCupr uint64,
) error {
	existingHistory := k.GetServiceComputeUnitsPerRelayHistoryForService(ctx, serviceId)
	if len(existingHistory) == 0 {
		if err := k.SetServiceComputeUnitsPerRelayAtHeight(ctx, 1, serviceId, prevCupr); err != nil {
			return err
		}
	}

	return k.SetServiceComputeUnitsPerRelayAtHeight(
		ctx,
		k.nextSessionStartHeight(ctx),
		serviceId,
		newCupr,
	)
}

// GetServiceComputeUnitsPerRelayHistoryForService returns all cupr history entries
// for a service.
func (k Keeper) GetServiceComputeUnitsPerRelayHistoryForService(
	ctx context.Context,
	serviceId string,
) []types.ServiceComputeUnitsPerRelayUpdate {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceHistoryPrefix := types.ServiceComputeUnitsPerRelayHistoryKeyPrefixForService(serviceId)
	historyStore := prefix.NewStore(store, serviceHistoryPrefix)

	iterator := historyStore.Iterator(nil, nil)
	defer iterator.Close()

	var history []types.ServiceComputeUnitsPerRelayUpdate
	for ; iterator.Valid(); iterator.Next() {
		var update types.ServiceComputeUnitsPerRelayUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &update)
		history = append(history, update)
	}

	return history
}

// GetAllServiceComputeUnitsPerRelayHistory returns all cupr history updates across
// all services. Primarily used for genesis export, debugging and testing.
func (k Keeper) GetAllServiceComputeUnitsPerRelayHistory(
	ctx context.Context,
) []types.ServiceComputeUnitsPerRelayUpdate {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	historyStore := prefix.NewStore(store, []byte(types.ServiceComputeUnitsPerRelayHistoryKeyPrefix))

	iterator := historyStore.Iterator(nil, nil)
	defer iterator.Close()

	var history []types.ServiceComputeUnitsPerRelayUpdate
	for ; iterator.Valid(); iterator.Next() {
		var update types.ServiceComputeUnitsPerRelayUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &update)
		history = append(history, update)
	}

	return history
}

// nextSessionStartHeight returns the start height of the session immediately after
// the one containing the current block height, using the live shared params. cupr
// changes are activated at this boundary so no in-flight session is affected.
func (k Keeper) nextSessionStartHeight(ctx context.Context) int64 {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	return sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)
}

// currentSessionStartHeight returns the start height of the session containing the
// current block height, using the live shared params. A newly created service records
// its initial cupr at this height so that its very first (possibly partial) session is
// pinned rather than resolving through the mutable live-cupr fallback. No claim can
// reference a session that started before the service existed, so this cannot pin a
// value for a pre-creation session.
func (k Keeper) currentSessionStartHeight(ctx context.Context) int64 {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	return sharedtypes.GetSessionStartHeight(&sharedParams, currentHeight)
}
