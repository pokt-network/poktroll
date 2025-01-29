package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// SetMorseAccountState set morseAccountState in the store
func (k Keeper) SetMorseAccountState(ctx context.Context, morseAccountState types.MorseAccountState) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountStateKey))
	b := k.cdc.MustMarshal(&morseAccountState)
	store.Set([]byte{0}, b)
}

// GetMorseAccountState returns morseAccountState
func (k Keeper) GetMorseAccountState(ctx context.Context) (val types.MorseAccountState, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountStateKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveMorseAccountState removes morseAccountState from the store
func (k Keeper) RemoveMorseAccountState(ctx context.Context) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.MorseAccountStateKey))
	store.Delete([]byte{0})
}
