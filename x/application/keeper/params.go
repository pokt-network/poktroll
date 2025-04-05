package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/application/types"
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

// GetParamsUpdates get all the module params updates history.
func (k Keeper) GetParamsUpdates(ctx context.Context) []*types.ParamsUpdate {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ParamsUpdateKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	paramsUpdates := make([]*types.ParamsUpdate, 0)
	for ; iterator.Valid(); iterator.Next() {
		var paramsUpdate types.ParamsUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &paramsUpdate)
		paramsUpdates = append(paramsUpdates, &paramsUpdate)
	}

	if len(paramsUpdates) == 0 {
		params := k.GetParams(ctx)
		paramsUpdates = append(paramsUpdates, &types.ParamsUpdate{
			Params:               params,
			EffectiveBlockHeight: 1,
		})
	}

	return paramsUpdates
}

// GetParamsAtHeight get the module params that are effective at a specific height.
func (k Keeper) GetParamsAtHeight(ctx context.Context, height int64) types.Params {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ParamsUpdateKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	var paramsAtHeight *types.ParamsUpdate
	for ; iterator.Valid(); iterator.Next() {
		var paramsUpdate types.ParamsUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &paramsUpdate)

		// Look for the most recent params update that is older or equal to the requested height.

		if paramsAtHeight == nil {
			paramsAtHeight = &paramsUpdate
			continue
		}

		// The paramsUpdate is in the future and not yet effective as of height, skip it.
		if paramsUpdate.EffectiveBlockHeight > uint64(height) {
			continue
		}

		// The paramsUpdate more recent than the current paramsAtHeight, set it.
		// This is the most recent params update that is effective at the requested height.
		if paramsUpdate.EffectiveBlockHeight >= paramsAtHeight.EffectiveBlockHeight {
			paramsAtHeight = &paramsUpdate
		}
	}

	// In case there are no params updates at all (i.e. only genesis params),
	// then set the params to the current ones.
	if paramsAtHeight == nil {
		currentParams := k.GetParams(ctx)
		paramsAtHeight = &types.ParamsUpdate{
			Params:               currentParams,
			EffectiveBlockHeight: 1,
		}
	}

	return paramsAtHeight.Params
}

// SetParamsUpdate stores a params update that will be effective at a specific block height.
func (k Keeper) SetParamsUpdate(ctx context.Context, paramsUpdate types.ParamsUpdate) error {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ParamsUpdateKeyPrefix))
	bz, err := k.cdc.Marshal(&paramsUpdate)
	if err != nil {
		return err
	}
	store.Set(types.ParamsUpdateKey(paramsUpdate.EffectiveBlockHeight), bz)

	return nil
}
