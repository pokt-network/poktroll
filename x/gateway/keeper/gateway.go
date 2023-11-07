package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// SetGateway set a specific gateway in the store from its index
func (k Keeper) SetGateway(ctx sdk.Context, gateway types.Gateway) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GatewayKeyPrefix))
	b := k.cdc.MustMarshal(&gateway)
	store.Set(types.GatewayKey(
		gateway.Address,
	), b)
}

// GetGateway returns a gateway from its index
func (k Keeper) GetGateway(
	ctx sdk.Context,
	address string,

) (val types.Gateway, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GatewayKeyPrefix))

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
	ctx sdk.Context,
	address string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GatewayKeyPrefix))
	store.Delete(types.GatewayKey(
		address,
	))
}

// GetAllGateway returns all gateways
func (k Keeper) GetAllGateway(ctx sdk.Context) (list []types.Gateway) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GatewayKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Gateway
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
