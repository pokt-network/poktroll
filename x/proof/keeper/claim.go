package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// UpsertClaim set a specific claim in the store from its index
func (k Keeper) UpsertClaim(ctx context.Context, claim types.Claim) {
	logger := k.Logger().With("method", "UpsertClaim")

	claimBz := k.cdc.MustMarshal(&claim)
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	sessionId := claim.GetSessionHeader().GetSessionId()
	primaryKey := types.ClaimPrimaryKey(sessionId, claim.SupplierAddress)
	primaryStore.Set(primaryKey, claimBz)

	logger.Info(fmt.Sprintf("upserted claim for supplier %s with primaryKey %s", claim.SupplierAddress, primaryKey))

	// Update the address index: supplierAddress -> [ClaimPrimaryKey]
	addressStoreIndex := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSupplierAddressPrefix))
	addressKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, primaryKey)
	addressStoreIndex.Set(addressKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed claim for supplier %s with primaryKey %s", claim.SupplierAddress, primaryKey))

	// Update the session end height index: sessionEndHeight -> [ClaimPrimaryKey]
	sessionHeightStoreIndex := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))
	sessionEndBlockHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	heightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndBlockHeight, primaryKey)
	sessionHeightStoreIndex.Set(heightKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed claim for supplier %s at session ending height %d", claim.SupplierAddress, sessionEndBlockHeight))
}

// GetClaim returns a claim from its index
func (k Keeper) GetClaim(ctx context.Context, sessionId, supplierAddr string) (claim types.Claim, found bool) {
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	return k.getClaimByPrimaryKey(ctx, primaryKey)
}

// RemoveClaim removes a claim from the store
func (k Keeper) RemoveClaim(ctx context.Context, sessionId, supplierAddr string) {
	logger := k.Logger().With("method", "RemoveClaim")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	// Check if the claim exists
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	claim, foundClaim := k.getClaimByPrimaryKey(ctx, primaryKey)
	if !foundClaim {
		logger.Error(fmt.Sprintf("trying to delete non-existent claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId))
		return
	}

	// Prepare the indices for deletion
	addressStoreIndex := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSupplierAddressPrefix))
	sessionHeightStoreIndex := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))

	addressKey := types.ClaimSupplierAddressKey(claim.GetSupplierAddress(), primaryKey)
	sessionEndBlockHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	heightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndBlockHeight, primaryKey)

	// Delete all the entries (primary store and secondary indices)
	primaryStore.Delete(primaryKey)
	addressStoreIndex.Delete(addressKey)
	sessionHeightStoreIndex.Delete(heightKey)

	logger.Info(fmt.Sprintf("deleted claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId))
}

// GetAllClaims returns all claim
func (k Keeper) GetAllClaims(ctx context.Context) (claims []types.Claim) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(primaryStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var claim types.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &claim)
		claims = append(claims, claim)
	}

	return
}

// getClaimByPrimaryKey is a helper that retrieves, if exists, the Claim associated with the key provided
func (k Keeper) getClaimByPrimaryKey(ctx context.Context, primaryKey []byte) (val types.Claim, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	b := primaryStore.Get(primaryKey)
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}
