package keeper

import (
	"context"
	"slices"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// GetParams retrieves all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz := store.Get(types.ParamsKey)
	if paramsBz == nil {
		return params
	}

	k.cdc.MustUnmarshal(paramsBz, &params)
	return params
}

// SetParams stores the proof parameters
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, paramsBz)

	return nil
}

// GetParamsUpdates retrieves the complete parameter update history for the module
func (k Keeper) GetParamsUpdates(ctx context.Context) []*types.ParamsUpdate {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ParamsUpdateKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	// Collect all parameter updates from the store
	paramsUpdates := make([]*types.ParamsUpdate, 0)
	for ; iterator.Valid(); iterator.Next() {
		var paramsUpdate types.ParamsUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &paramsUpdate)
		paramsUpdates = append(paramsUpdates, &paramsUpdate)
	}

	// Sort updates chronologically by effective height
	slices.SortFunc(paramsUpdates, func(a, b *types.ParamsUpdate) int {
		return int(a.ActivationHeight - b.ActivationHeight)
	})

	return paramsUpdates
}

// GetParamsAtHeight retrieves the module parameters effective at a specific block height
func (k Keeper) GetParamsAtHeight(ctx context.Context, queryHeight int64) types.Params {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ParamsUpdateKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	var paramsAtHeight *types.ParamsUpdate
	for ; iterator.Valid(); iterator.Next() {
		var paramsUpdate types.ParamsUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &paramsUpdate)

		// Initialize with first parameter update found
		if paramsAtHeight == nil {
			paramsAtHeight = &paramsUpdate
			continue
		}

		// Skip updates from the future (activation height > query height)
		if paramsUpdate.ActivationHeight > queryHeight {
			continue
		}

		// Select the most recent parameter update that's active at the query height
		// (has the highest activation height that's still ≤ query height)
		if paramsUpdate.ActivationHeight >= paramsAtHeight.ActivationHeight {
			paramsAtHeight = &paramsUpdate
		}
	}

	return paramsAtHeight.Params
}

// SetParamsUpdate stores a parameter update to become effective at a specific block height
func (k Keeper) SetParamsUpdate(ctx context.Context, paramsUpdate types.ParamsUpdate) error {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ParamsUpdateKeyPrefix))
	bz, err := k.cdc.Marshal(&paramsUpdate)
	if err != nil {
		return err
	}
	store.Set(types.IntKey(paramsUpdate.ActivationHeight), bz)

	return nil
}

// SetInitialParams stores the initial parameters and the update history to
// become effective at block height 1
func (k Keeper) SetInitialParams(ctx context.Context, params types.Params) error {
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	paramsUpdate := types.ParamsUpdate{
		Params:             params,
		ActivationHeight:   1,
		DeactivationHeight: 0,
	}

	if err := k.SetParamsUpdate(ctx, paramsUpdate); err != nil {
		return err
	}

	return nil
}
