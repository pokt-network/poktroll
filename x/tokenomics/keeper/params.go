package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	if k.cachedParams != nil {
		return *k.cachedParams
	}
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz := store.Get(types.ParamsKey)
	if paramsBz == nil {
		return params
	}

	k.cdc.MustUnmarshal(paramsBz, &params)
	k.cachedParams = &params
	return params
}

// SetParams set the params
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	k.cachedParams = &params
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, paramsBz)

	return nil
}

func (k Keeper) ResetCache() {
	k.cachedParams = nil
	k.applicationKeeper.ResetCache()
	k.supplierKeeper.ResetCache()
	k.sharedKeeper.ResetCache()
	k.sessionKeeper.ResetCache()
	k.proofKeeper.ResetCache()
	k.serviceKeeper.ResetCache()
}
