package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// SetGateway set a specific gateway in the store from its index
func (k Keeper) SetGateway(ctx context.Context, gateway gateway.Gateway) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	gatewayBz := k.cdc.MustMarshal(&gateway)
	store.Set(types.GatewayKey(
		gateway.Address,
	), gatewayBz)
}

// GetGateway returns a gateway from its index
func (k Keeper) GetGateway(
	ctx context.Context,
	address string,
) (gateway gateway.Gateway, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))

	gatewayBz := store.Get(types.GatewayKey(
		address,
	))
	if gatewayBz == nil {
		return gateway, false
	}

	k.cdc.MustUnmarshal(gatewayBz, &gateway)
	return gateway, true
}

// RemoveGateway removes a gateway from the store
func (k Keeper) RemoveGateway(
	ctx context.Context,
	address string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	store.Delete(types.GatewayKey(address))
}

// GetAllGateways returns all gateway
func (k Keeper) GetAllGateways(ctx context.Context) (gateways []gateway.Gateway) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var gateway gateway.Gateway
		k.cdc.MustUnmarshal(iterator.Value(), &gateway)
		gateways = append(gateways, gateway)
	}

	return
}
