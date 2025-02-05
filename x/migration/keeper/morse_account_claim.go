package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// SetMorseAccountClaim set a specific morseAccountClaim in the store from its index
func (k Keeper) SetMorseAccountClaim(ctx context.Context, morseAccountClaim types.MorseAccountClaim) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountClaimKeyPrefix))
	b := k.cdc.MustMarshal(&morseAccountClaim)
	store.Set(types.MorseAccountClaimKey(
		morseAccountClaim.MorseSrcAddress,
	), b)
}

// GetMorseAccountClaim returns a morseAccountClaim from its index
func (k Keeper) GetMorseAccountClaim(
	ctx context.Context,
	morseSrcAddress string,

) (val types.MorseAccountClaim, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountClaimKeyPrefix))

	b := store.Get(types.MorseAccountClaimKey(
		morseSrcAddress,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveMorseAccountClaim removes a morseAccountClaim from the store
func (k Keeper) RemoveMorseAccountClaim(
	ctx context.Context,
	morseSrcAddress string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountClaimKeyPrefix))
	store.Delete(types.MorseAccountClaimKey(
		morseSrcAddress,
	))
}

// GetAllMorseAccountClaim returns all morseAccountClaim
func (k Keeper) GetAllMorseAccountClaim(ctx context.Context) (list []types.MorseAccountClaim) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountClaimKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.MorseAccountClaim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
