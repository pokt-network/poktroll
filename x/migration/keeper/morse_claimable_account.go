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
	morseClaimableAccountBz := k.cdc.MustMarshal(&morseClaimableAccount)
	store.Set(types.MorseClaimableAccountKey(
		morseClaimableAccount.MorseSrcAddress,
	), morseClaimableAccountBz)
}

// GetMorseClaimableAccount returns a morseClaimableAccount from its index
func (k Keeper) GetMorseClaimableAccount(
	ctx context.Context,
	address string,

) (morseClaimableAccount types.MorseClaimableAccount, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))

	morseClaimableAccountBz := store.Get(types.MorseClaimableAccountKey(
		address,
	))
	if morseClaimableAccountBz == nil {
		return morseClaimableAccount, false
	}

	k.cdc.MustUnmarshal(morseClaimableAccountBz, &morseClaimableAccount)
	return morseClaimableAccount, true
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

// GetAllMorseClaimableAccounts returns all morseClaimableAccount
func (k Keeper) GetAllMorseClaimableAccounts(ctx context.Context) (list []types.MorseClaimableAccount) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseClaimableAccountKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var morseClaimableAccount types.MorseClaimableAccount
		k.cdc.MustUnmarshal(iterator.Value(), &morseClaimableAccount)
		list = append(list, morseClaimableAccount)
	}

	return
}

// ImportFromMorseAccountState imports the MorseClaimableAccounts from the given MorseAccountState.
func (k Keeper) ImportFromMorseAccountState(
	ctx context.Context,
	morseAccountState *types.MorseAccountState,
) {
	for _, morseAccount := range morseAccountState.Accounts {
		k.SetMorseClaimableAccount(ctx, *morseAccount)
	}
}
