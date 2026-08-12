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
	gatewayBz := k.cdc.MustMarshal(&gateway)
	store.Set(types.GatewayKey(
		gateway.Address,
	), gatewayBz)
}

// GetGateway returns a gateway from its index
func (k Keeper) GetGateway(
	ctx context.Context,
	address string,
) (gateway types.Gateway, found bool) {
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

// GetAllGatewayLifecycles returns every gateway in state decoded WITHOUT its card.
//
// Prefer this over GetAllGateways on any path that only needs lifecycle state. A card
// can be up to MaxServiceMetadataSizeBytes (256 KiB), and the two EndBlockers that scan
// every gateway -- x/application's EndBlockerAutoUndelegateFromUnbondingGateways (EVERY
// block) and x/gateway's EndBlockerUnbondGateways (every session end) -- never read one.
// Decoding full records there would have every validator allocate
// (carded gateways x card size) per block for data it discards, and that work is not
// gas-metered, so nothing throttles it: staking gateways with maxed cards (a refundable
// one-time cost) would impose an unbounded recurring cost on the whole network.
//
// GatewayLifecycle mirrors Gateway's leading field numbers, so the stored bytes decode
// directly; the card (field 4) is skipped by the generated unmarshaller's index
// arithmetic and never copied. This reads the same bytes from the store as
// GetAllGateways, so gas is identical -- only the decode is cheaper.
func (k Keeper) GetAllGatewayLifecycles(ctx context.Context) (gateways []types.GatewayLifecycle) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var gateway types.GatewayLifecycle
		k.cdc.MustUnmarshal(iterator.Value(), &gateway)
		gateways = append(gateways, gateway)
	}

	return
}

// GetAllGateways returns every gateway in state, fully hydrated (cards included).
// On hot iteration paths that do not need the card, use GetAllGatewayLifecycles.
func (k Keeper) GetAllGateways(ctx context.Context) (gateways []types.Gateway) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.GatewayKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var gateway types.Gateway
		k.cdc.MustUnmarshal(iterator.Value(), &gateway)
		gateways = append(gateways, gateway)
	}

	return
}
