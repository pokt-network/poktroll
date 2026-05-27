package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/shared/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// SetParamsAtHeight stores a snapshot of session params with their effective height.
// This enables historical lookups of params that were active at a given block height.
func (k Keeper) SetParamsAtHeight(ctx context.Context, effectiveHeight int64, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	paramsUpdate := types.ParamsUpdate{
		EffectiveHeight: effectiveHeight,
		Params:          &params,
	}

	bz, err := k.cdc.Marshal(&paramsUpdate)
	if err != nil {
		return err
	}

	key := types.ParamsHistoryKey(effectiveHeight)
	store.Set(key, bz)

	return nil
}

// GetParamsAtHeight returns the session params that were effective at the given height.
// It finds the most recent params entry where effective_height <= queryHeight.
// If no historical params exist, it returns the current params (backwards compatible).
func (k Keeper) GetParamsAtHeight(ctx context.Context, queryHeight int64) types.Params {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	historyStore := prefix.NewStore(store, types.ParamsHistoryKeyPrefix)

	// Create an iterator that goes from the query height backwards to find
	// the most recent params that were effective at or before the query height.
	// We use a reverse iterator with end key = queryHeight+1 (exclusive upper bound).
	endKey := make([]byte, 8)
	binary.BigEndian.PutUint64(endKey, uint64(queryHeight+1))

	iterator := historyStore.ReverseIterator(nil, endKey)
	defer iterator.Close()

	if iterator.Valid() {
		var paramsUpdate types.ParamsUpdate
		// Defensive: a corrupted history entry (e.g., partial write, downgrade
		// from a newer schema, on-disk bit rot) must not halt the chain via
		// MustUnmarshal. Log + fall through to GetParams; resolving to live
		// params is the same behavior as a missing entry, which downstream
		// callers already tolerate.
		if err := k.cdc.Unmarshal(iterator.Value(), &paramsUpdate); err != nil {
			k.logger.Error(fmt.Sprintf(
				"GetParamsAtHeight: failed to unmarshal params history entry at queryHeight=%d: %v; falling back to live params",
				queryHeight, err,
			))
		} else if paramsUpdate.Params != nil {
			return *paramsUpdate.Params
		}
	}

	// Fallback: If no historical params found, return current params.
	// This maintains backwards compatibility for chains that haven't
	// recorded any param history yet.
	return k.GetParams(ctx)
}

// GetParamsHistoryEntry returns the params recorded with an effective height EXACTLY equal
// to effectiveHeight, and whether such an entry exists. Unlike GetParamsAtHeight (which
// finds the most recent entry <= a height), this is an exact-key lookup used by the
// EndBlocker to detect an epoch that becomes effective at precisely the current block
// (#543 anchored grid, Option B promotion).
func (k Keeper) GetParamsHistoryEntry(ctx context.Context, effectiveHeight int64) (types.Params, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	bz := store.Get(types.ParamsHistoryKey(effectiveHeight))
	if bz == nil {
		return types.Params{}, false
	}

	var paramsUpdate types.ParamsUpdate
	// Defensive: a corrupted history entry must not halt the chain. Treat an
	// unmarshal failure the same as a missing entry — callers (the EndBlocker
	// promotion path) already handle "no entry at this height" by falling
	// through to the live params, so the safest recovery is to surface this as
	// "no entry" rather than panic the chain on the deferred-promotion block.
	if err := k.cdc.Unmarshal(bz, &paramsUpdate); err != nil {
		k.logger.Error(fmt.Sprintf(
			"GetParamsHistoryEntry: failed to unmarshal params history entry at effectiveHeight=%d: %v; treating as missing",
			effectiveHeight, err,
		))
		return types.Params{}, false
	}
	if paramsUpdate.Params == nil {
		return types.Params{}, false
	}

	return *paramsUpdate.Params, true
}

// HasParamsHistory returns true if any params history entries exist.
// This is used to efficiently check if history needs initialization without
// the O(n) cost of GetAllParamsHistory.
func (k Keeper) HasParamsHistory(ctx context.Context) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	historyStore := prefix.NewStore(store, types.ParamsHistoryKeyPrefix)
	iterator := historyStore.Iterator(nil, nil)
	defer iterator.Close()
	return iterator.Valid()
}

// IterateParamsHistoryReverse iterates params history entries in reverse order of
// effective_height starting from the largest entry with effective_height <= fromHeight.
// The callback fn is invoked for each entry; returning stop=true halts iteration.
//
// Used by callers that need to resolve, for each historical params epoch, what
// claim/session timing math applies — e.g. settlement walking recent epochs to
// find every candidate sessionEndHeight whose proof window closes at the current
// block (cross-session window-offset orphan class, O2).
func (k Keeper) IterateParamsHistoryReverse(
	ctx context.Context,
	fromHeight int64,
	fn func(effectiveHeight int64, params types.Params) (stop bool),
) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	historyStore := prefix.NewStore(store, types.ParamsHistoryKeyPrefix)

	// end key is exclusive; +1 so an entry at effective_height == fromHeight is included.
	endKey := make([]byte, 8)
	binary.BigEndian.PutUint64(endKey, uint64(fromHeight+1))

	iterator := historyStore.ReverseIterator(nil, endKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var paramsUpdate types.ParamsUpdate
		// Defensive: skip-and-log on corrupted history entries instead of halting
		// via MustUnmarshal. Same rationale as GetParamsAtHeight + GetParamsHistoryEntry —
		// a corrupted entry must not halt the chain at settlement (the only caller of
		// this iterator is the O2 cross-session candidate scan in
		// candidateSessionEndHeightsForLiveParams). Skipping yields a degraded scan
		// — that epoch's candidates are missed — which is observable and recoverable;
		// halting is not.
		if err := k.cdc.Unmarshal(iterator.Value(), &paramsUpdate); err != nil {
			k.logger.Error(fmt.Sprintf(
				"IterateParamsHistoryReverse: failed to unmarshal params history entry at fromHeight=%d: %v; skipping entry",
				fromHeight, err,
			))
			continue
		}
		if paramsUpdate.Params == nil {
			continue
		}
		if fn(paramsUpdate.EffectiveHeight, *paramsUpdate.Params) {
			return
		}
	}
}

// GetAllParamsHistory returns all historical session params updates.
// This is primarily used for genesis export and debugging.
func (k Keeper) GetAllParamsHistory(ctx context.Context) []types.ParamsUpdate {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	historyStore := prefix.NewStore(store, types.ParamsHistoryKeyPrefix)

	iterator := historyStore.Iterator(nil, nil)
	defer iterator.Close()

	var history []types.ParamsUpdate
	for ; iterator.Valid(); iterator.Next() {
		var paramsUpdate types.ParamsUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &paramsUpdate)
		history = append(history, paramsUpdate)
	}

	return history
}
