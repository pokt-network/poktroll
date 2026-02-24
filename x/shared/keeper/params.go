package keeper

import (
	"context"
	"encoding/binary"

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
		k.cdc.MustUnmarshal(iterator.Value(), &paramsUpdate)
		if paramsUpdate.Params != nil {
			return *paramsUpdate.Params
		}
	}

	// Fallback: If no historical params found, return current params.
	// This maintains backwards compatibility for chains that haven't
	// recorded any param history yet.
	return k.GetParams(ctx)
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
