package keeper

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// UpsertClaim inserts or updates a specific claim in the store by index.
func (k Keeper) UpsertClaim(ctx sdk.Context, claim types.Claim) {
	logger := k.Logger(ctx).With("method", "UpsertClaim")

	claimBz := k.cdc.MustMarshal(&claim)
	parentStore := ctx.KVStore(k.storeKey)

	// Update the primary store: ClaimPrimaryKey -> ClaimObject
	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	sessionId := claim.GetSessionHeader().GetSessionId()
	primaryKey := types.ClaimPrimaryKey(sessionId, claim.SupplierAddress)
	primaryStore.Set(primaryKey, claimBz)

	logger.Info(fmt.Sprintf("upserted claim for supplier %s with primaryKey %s", claim.SupplierAddress, primaryKey))

	// Update the address index: supplierAddress -> [ClaimPrimaryKey]
	addressStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimSupplierAddressPrefix))
	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, primaryKey)
	addressStoreIndex.Set(addressKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed claim for supplier %s with primaryKey %s", claim.SupplierAddress, primaryKey))

	// Update the session end height index: sessionEndHeight -> [ClaimPrimaryKey]
	sessionHeightStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))
	sessionEndBlockHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	heightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndBlockHeight, primaryKey)
	sessionHeightStoreIndex.Set(heightKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed claim for supplier %s at session ending height %d", claim.SupplierAddress, sessionEndBlockHeight))
}

// RemoveClaim removes a claim from the store
func (k Keeper) RemoveClaim(ctx sdk.Context, sessionId, supplierAddr string) {
	logger := k.Logger(ctx).With("method", "RemoveClaim")

	parentStore := ctx.KVStore(k.storeKey)
	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	// Check if the claim exists
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	claim, foundClaim := k.getClaimByPrimaryKey(ctx, primaryKey)
	if !foundClaim {
		logger.Error(fmt.Sprintf("trying to delete non-existent claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId))
		return
	}

	// Prepare the indices for deletion
	addressStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimSupplierAddressPrefix))
	sessionHeightStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))

	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, primaryKey)
	sessionEndBlockHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	heightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndBlockHeight, primaryKey)

	// Delete all the entries (primary store and secondary indices)
	primaryStore.Delete(primaryKey)
	addressStoreIndex.Delete(addressKey)
	sessionHeightStoreIndex.Delete(heightKey)

	logger.Info(fmt.Sprintf("deleted claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId))
}

// GetClaim returns a Claim given a SessionId & SupplierAddr
func (k Keeper) GetClaim(ctx sdk.Context, sessionId, supplierAddr string) (val types.Claim, found bool) {
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	return k.getClaimByPrimaryKey(ctx, primaryKey)
}

// GetAllClaims returns all claim
func (k Keeper) GetAllClaims(ctx sdk.Context) (claims []types.Claim) {
	primaryStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(primaryStore, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		claims = append(claims, val)
	}

	return
}

// GetClaimsByAddress returns all claims for a given supplier address
func (k Keeper) GetClaimsByAddress(ctx sdk.Context, address string) (claims []types.Claim) {
	addressStoreIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimSupplierAddressPrefix))

	iterator := sdk.KVStorePrefixIterator(addressStoreIndex, []byte(address))
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

// GetClaimsByAddress returns all claims whose session ended at the given block height
func (k Keeper) GetClaimsByHeight(ctx sdk.Context, height uint64) (claims []types.Claim) {
	sessionHeightStoreIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimSessionEndHeightPrefix))

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, height)

	iterator := sdk.KVStorePrefixIterator(sessionHeightStoreIndex, heightBz)
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

// GetClaimsByAddress returns all claims matching the given session id
func (k Keeper) GetClaimsBySession(ctx sdk.Context, sessionId string) (claims []types.Claim) {
	sessionIdStoreIndex := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	iterator := sdk.KVStorePrefixIterator(sessionIdStoreIndex, []byte(sessionId))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		claims = append(claims, val)
	}

	return claims
}

// getClaimByPrimaryKey is a helper that retrieves, if exists, the Claim associated with the key provided
func (k Keeper) getClaimByPrimaryKey(ctx sdk.Context, primaryKey []byte) (val types.Claim, found bool) {
	primaryStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	b := primaryStore.Get(primaryKey)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}
