package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// SetClaim set a specific claim in the store from its index
func (k Keeper) SetClaim(ctx context.Context, claim types.Claim) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimKeyPrefix))
	b := k.cdc.MustMarshal(&claim)
	store.Set(types.ClaimKey(
		claim.Index,
	), b)
}

// GetClaim returns a claim from its index
func (k Keeper) GetClaim(
	ctx context.Context,
	index string,

) (val types.Claim, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimKeyPrefix))

	b := store.Get(types.ClaimKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveClaim removes a claim from the store
func (k Keeper) RemoveClaim(
	ctx context.Context,
	index string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimKeyPrefix))
	store.Delete(types.ClaimKey(
		index,
	))
}

// GetAllClaim returns all claim
func (k Keeper) GetAllClaim(ctx context.Context) (list []types.Claim) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
