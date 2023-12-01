package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SetProof set a specific proof in the store from its index
func (k Keeper) SetProof(ctx sdk.Context, proof types.Proof) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))
	b := k.cdc.MustMarshal(&proof)
	store.Set(types.ProofKey(
		proof.Index,
	), b)
}

// GetProof returns a proof from its index
func (k Keeper) GetProof(
	ctx sdk.Context,
	index string,

) (val types.Proof, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))

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
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))
	store.Delete(types.ProofKey(
		index,
	))
}

// GetAllProofs returns all proof
func (k Keeper) GetAllProofs(ctx sdk.Context) (list []types.Proof) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Proof
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
