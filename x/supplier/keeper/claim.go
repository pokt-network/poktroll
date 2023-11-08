package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// SetClaim set a specific claim in the store from its index
func (k Keeper) SetClaim(ctx sdk.Context, claim types.Claim) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimKeyPrefix))
	b := k.cdc.MustMarshal(&claim)
	store.Set(types.ClaimKey(
		claim.Index,
	), b)
}

// GetClaim returns a claim from its index
func (k Keeper) GetClaim(
	ctx sdk.Context,
	index string,

) (val types.Claim, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimKeyPrefix))

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
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimKeyPrefix))
	store.Delete(types.ClaimKey(
		index,
	))
}

// GetAllClaims returns all claim
func (k Keeper) GetAllClaims(ctx sdk.Context) (list []types.Claim) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
