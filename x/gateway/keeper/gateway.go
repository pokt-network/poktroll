package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// SetGateway set a specific gateway in the store from its index
func (k Keeper) SetGateway(ctx context.Context, gateway types.Gateway) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	b := k.cdc.MustMarshal(&gateway)
	store.Set(types.GatewayKey(
		gateway.Address,
	), b)
}

// GetGateway returns a gateway from its index
func (k Keeper) GetGateway(
	ctx context.Context,
	address string,

) (val types.Gateway, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))

	b := store.Get(types.GatewayKey(
		address,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveGateway removes a gateway from the store
func (k Keeper) RemoveGateway(
	ctx context.Context,
	address string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	store.Delete(types.GatewayKey(
		address,
	))
}

// GetAllGateway returns all gateway
func (k Keeper) GetAllGateway(ctx context.Context) (list []types.Gateway) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Gateway
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
