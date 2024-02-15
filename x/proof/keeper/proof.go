package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// SetProof set a specific proof in the store from its index
func (k Keeper) SetProof(ctx context.Context, proof types.Proof) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofKeyPrefix))
	b := k.cdc.MustMarshal(&proof)
	store.Set(types.ProofKey(
		proof.Index,
	), b)
}

// GetProof returns a proof from its index
func (k Keeper) GetProof(
	ctx context.Context,
	index string,

) (val types.Proof, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofKeyPrefix))

	b := store.Get(types.ProofKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveProof removes a proof from the store
func (k Keeper) RemoveProof(
	ctx context.Context,
	index string,

) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofKeyPrefix))
	store.Delete(types.ProofKey(
		index,
	))
}

// GetAllProof returns all proof
func (k Keeper) GetAllProof(ctx context.Context) (list []types.Proof) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Proof
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
