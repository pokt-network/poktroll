package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// SetMorseClaimableAccount set a specific morseClaimableAccount in the store from its index
func (k Keeper) SetMorseClaimableAccount(ctx context.Context, morseClaimableAccount types.MorseClaimableAccount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))
	b := k.cdc.MustMarshal(&morseClaimableAccount)
	store.Set(types.MorseClaimableAccountKey(
		morseClaimableAccount.Address.String(),
	), b)
}

// GetMorseClaimableAccount returns a morseClaimableAccount from its index
func (k Keeper) GetMorseClaimableAccount(
	ctx context.Context,
	address string,

) (val types.MorseClaimableAccount, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))

	b := store.Get(types.MorseClaimableAccountKey(
		address,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveMorseClaimableAccount removes a morseClaimableAccount from the store
func (k Keeper) RemoveMorseClaimableAccount(
	ctx context.Context,
	address string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))
	store.Delete(types.MorseClaimableAccountKey(
		address,
	))
}

// GetAllMorseClaimableAccount returns all morseClaimableAccount
func (k Keeper) GetAllMorseClaimableAccount(ctx context.Context) (list []types.MorseClaimableAccount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.MorseClaimableAccount
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
