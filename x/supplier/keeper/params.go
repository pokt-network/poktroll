package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/supplier/types"
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

// SetParams sets the params. When they enable or change the service cap, it
// schedules a service cap sweep, so every params write (genesis, upgrade
// handlers, governance) re-checks existing suppliers against the new cap.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.setParams(ctx, k.GetParams(ctx), params)
}

// setParams is SetParams for callers that already hold the stored params.
func (k Keeper) setParams(ctx context.Context, paramsBefore, params types.Params) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	paramsBz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, paramsBz)

	if params.ServiceCapEnabled() && serviceCapParamsChanged(paramsBefore, params) {
		k.scheduleServiceCapSweep(ctx)
	}

	return nil
}
