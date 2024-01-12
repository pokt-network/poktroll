package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// UpsertProof inserts or updates a specific proof in the store by index.
func (k Keeper) UpsertProof(ctx sdk.Context, proof types.Proof) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))
	b := k.cdc.MustMarshal(&proof)
	// TODO_NEXT(@bryanchriswhite #141): Refactor keys to support multiple indices.
	store.Set(types.ProofKey(
		proof.GetSessionHeader().GetSessionId(),
	), b)
}

// GetProof returns a proof from its index
func (k Keeper) GetProof(
	ctx sdk.Context,
	sessionId string,
) (val types.Proof, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))

	// TODO_NEXT(@bryanchriswhite #141): Refactor proof keys to support multiple indices.
	b := store.Get(types.ProofKey(
		sessionId,
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
	// TODO_NEXT(@bryanchriswhite #141): Refactor proof keys to support multiple indices.
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofKeyPrefix))
	// TODO_NEXT(@bryanchriswhite #141): Refactor proof keys to support multiple indices.
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
