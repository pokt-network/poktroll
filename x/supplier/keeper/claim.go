package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// InsertClaim adds a claim to the store
func (k Keeper) InsertClaim(ctx sdk.Context, claim types.Claim) {
	claimBz := k.cdc.MustMarshal(&claim)
	parentStore := ctx.KVStore(k.storeKey)

	// Update the primary store -
	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	primaryKey := types.ClaimPrimaryKey(claim.SessionId, claim.SupplierAddress)
	primaryStore.Set(primaryKey, claimBz)

	// Update the session index
	// TODO

	// Update the height index
	// TODO

	// Update the address index
	addressStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimAddressPrefix))
	addressClaimCount := k.getCount(ctx, addressStoreIndex)
	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, addressClaimCount)
	addressStoreIndex.Set(addressKey, primaryKey)
	k.setCount(ctx, addressStoreIndex, addressClaimCount+1)
}

// GetClaim returns a claim given a sessionId & supplierAddr
func (k Keeper) GetClaim(ctx sdk.Context, sessionId, supplierAddr string) (val types.Claim, found bool) {
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	return k.getClaimByPrimaryKey(ctx, primaryKey)
}

// GetAllClaims returns all claim
func (k Keeper) GetAllClaims(ctx sdk.Context) (claims []types.Claim) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		claims = append(claims, val)
	}

	return
}

// When retrieving by address:
func (k Keeper) GetClaimsByAddress(ctx sdk.Context, address sdk.AccAddress) (claims []types.Claim) {
	addressStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimAddressPrefix))

	iterator := sdk.KVStorePrefixIterator(addressStore, []byte(address))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		primaryKey := iterator.Value()
		claim, claimFound := k.getClaimByPrimaryKey(ctx, primaryKey)
		if claimFound {
			claims = append(claims, claim)
		}
	}

	return claims
}

func (k Keeper) getClaimByPrimaryKey(ctx sdk.Context, primaryKey []byte) (val types.Claim, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	b := store.Get(primaryKey)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) getCount(ctx sdk.Context, store prefix.Store) uint64 {
	bz := store.Get(types.CountKey)
	if bz == nil {
		return 0 // Count doesn't exist: no element
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) setCount(ctx sdk.Context, store prefix.Store, count uint64) {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.CountKey, bz)
}
