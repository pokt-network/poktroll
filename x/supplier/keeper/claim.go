package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// InsertClaim adds a claim to the store
func (k Keeper) InsertClaim(ctx sdk.Context, claim types.Claim) {
	logger := k.Logger(ctx).With("method", "InsertClaim")

	claimBz := k.cdc.MustMarshal(&claim)
	parentStore := ctx.KVStore(k.storeKey)

	// Update the primary store: ClaimPrimaryKey -> ClaimObject
	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	primaryKey := types.ClaimPrimaryKey(claim.SessionId, claim.SupplierAddress)
	primaryStore.Set(primaryKey, claimBz)

	logger.Info("inserted claim with primaryKey %s", primaryKey)

	// Update the address index: supplierAddress -> [ClaimPrimaryKey]
	addressStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimAddressPrefix))
	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, primaryKey)
	addressStoreIndex.Set(addressKey, primaryKey)

	// TODO: Index by sessionId
	// TODO: Index by sessionEndHeight
}

// RemoveClaim removes a claim from the store
func (k Keeper) RemoveClaim(ctx sdk.Context, sessionId, supplierAddr string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	claim, foundClaim := k.getClaimByPrimaryKey(ctx, primaryKey)
	if !foundClaim {
		k.Logger(ctx).Error("trying to delete non-existent claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId)
	}

	addressStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimAddressPrefix))
	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, primaryKey)

	addressStore.Delete(addressKey)
	store.Delete(primaryKey)
}

// GetClaim returns a Claim given a SessionId & SupplierAddr
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

// GetClaimsByAddress returns all claims for a given address
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

// getClaimByPrimaryKey is a helper that retrieves, if exists, the Claim associated with the key provided
func (k Keeper) getClaimByPrimaryKey(ctx sdk.Context, primaryKey []byte) (val types.Claim, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	b := store.Get(primaryKey)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}
