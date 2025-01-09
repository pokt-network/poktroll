package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	if k.cache.Params != nil {
		k.logger.Info("-----Tokenomics params cache hit-----")
		return *k.cache.Params
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz := store.Get(types.ParamsKey)
	if paramsBz == nil {
		return params
	}

	k.cdc.MustUnmarshal(paramsBz, &params)
	k.cache.Params = &params
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	k.cache.Params = &params
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, paramsBz)

	return nil
}

func (k Keeper) ClearCache() {
	k.cache.Clear()
	k.applicationKeeper.ClearCache()
	k.supplierKeeper.ClearCache()
	k.sharedKeeper.ClearCache()
	k.sessionKeeper.ClearCache()
	k.proofKeeper.ClearCache()
	k.serviceKeeper.ClearCache()
}
