package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz := store.Get(types.ParamsKey)
	if paramsBz == nil {
		return params
	}

	k.cdc.MustUnmarshal(paramsBz, &params)
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, paramsBz)

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

// RecordParamsHistory ensures session params history is properly tracked.
// It initializes history with current params if needed, then records new params
// with their effective height (next session start).
func (k Keeper) RecordParamsHistory(ctx context.Context, newParams types.Params) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get the OLD params before we update (these are the currently effective params)
	oldParams := k.GetParams(ctx)

	// Check if history is empty (first param update since genesis or upgrade)
	history := k.GetAllParamsHistory(ctx)
	if len(history) == 0 {
		// Initialize history with the current (old) params at the current height.
		// We use current height rather than height 1 because we can only vouch for
		// the params we know now - not what they may have been at genesis.
		// For heights before this, GetParamsAtHeight falls back to current params.
		if err := k.SetParamsAtHeight(ctx, currentHeight, oldParams); err != nil {
			return fmt.Errorf("failed to initialize session params history: %w", err)
		}
	}

	// Calculate when the new params become effective: start of next session.
	// We need the shared params to calculate the session boundary.
	sharedParams := k.sharedKeeper.GetParamsAtHeight(ctx, currentHeight)

	currentSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	nextSessionStartHeight := currentSessionEndHeight + 1

	// Store the new params with their effective height.
	if err := k.SetParamsAtHeight(ctx, nextSessionStartHeight, newParams); err != nil {
		return fmt.Errorf("failed to record new session params: %w", err)
	}

	return nil
}
