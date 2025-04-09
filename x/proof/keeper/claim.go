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
	primaryKey := types.ClaimPrimaryKey(sessionId, claim.SupplierOperatorAddress)
	primaryStore.Set(primaryKey, claimBz)
	logger.Info(fmt.Sprintf("upserted claim for supplier %s with primaryKey %s", claim.SupplierOperatorAddress, primaryKey))

	// Update the address index: supplierOperatorAddress -> [ClaimPrimaryKey]
	supplierOperatorAddrStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSupplierOperatorAddressPrefix))
	supplierOperatorAddrKey := types.ClaimSupplierOperatorAddressKey(claim.SupplierOperatorAddress, primaryKey)
	supplierOperatorAddrStore.Set(supplierOperatorAddrKey, primaryKey)
	logger.Info(fmt.Sprintf("indexed claim for supplier %s with primaryKey %s", claim.SupplierOperatorAddress, primaryKey))

	// Update the session end height index: sessionEndHeight -> [ClaimPrimaryKey]
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))
	sessionEndHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	sessionEndHeightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)
	sessionEndHeightStore.Set(sessionEndHeightKey, primaryKey)
	logger.Info(fmt.Sprintf("indexed claim for supplier %s at session ending height %d", claim.SupplierOperatorAddress, sessionEndHeight))
}

// GetClaim returns a claim from its index
func (k Keeper) GetClaim(ctx context.Context, sessionId, supplierOperatorAddr string) (_ types.Claim, isClaimFound bool) {
	return k.getClaimByPrimaryKey(ctx, types.ClaimPrimaryKey(sessionId, supplierOperatorAddr))
}

// RemoveClaim removes a claim from the store
func (k Keeper) RemoveClaim(ctx context.Context, sessionId, supplierOperatorAddr string) {
	logger := k.Logger().With("method", "RemoveClaim")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	// Check if the claim exists
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierOperatorAddr)
	foundClaim, isClaimFound := k.getClaimByPrimaryKey(ctx, primaryKey)
	if !isClaimFound {
		logger.Error(fmt.Sprintf("trying to delete non-existent claim with primary key %s for supplier %s and session %s", primaryKey, supplierOperatorAddr, sessionId))
		return
	}

	// Prepare the indices for deletion
	supplierOperatorAddrStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSupplierOperatorAddressPrefix))
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))

	supplierOperatorAddrKey := types.ClaimSupplierOperatorAddressKey(foundClaim.GetSupplierOperatorAddress(), primaryKey)
	sessionEndHeight := foundClaim.GetSessionHeader().GetSessionEndBlockHeight()
	sessionEndHeightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)

	// Delete all the entries (primary store and secondary indices)
	primaryStore.Delete(primaryKey)
	supplierOperatorAddrStore.Delete(supplierOperatorAddrKey)
	sessionEndHeightStore.Delete(sessionEndHeightKey)

	logger.Info(fmt.Sprintf("deleted claim with primary key %s for supplier %s and session %s", primaryKey, supplierOperatorAddr, sessionId))
}

// GetClaimsBySessionEndHeightIterator returns an iterator over all claims indexed by session end height
// The iterator will return all claims that have the same session end height
func (k Keeper) GetSessionEndHeightClaimsIterator(ctx context.Context, sessionEndHeight int64) *types.ClaimsIterator {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))
	sessionEndHeightPrefix := types.ClaimSupplierEndSessionHeightKey(sessionEndHeight, []byte{})
	iterator := storetypes.KVStorePrefixIterator(sessionEndHeightStore, sessionEndHeightPrefix)

	claimsIterator := types.NewClaimsIterator(iterator, k.cdc)

	return claimsIterator
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

	return claims
}

// getClaimByPrimaryKey is a helper that retrieves, if exists, the Claim associated with the key provided
func (k Keeper) getClaimByPrimaryKey(ctx context.Context, primaryKey []byte) (claim types.Claim, isClaimFound bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	claimBz := primaryStore.Get(primaryKey)

	if claimBz == nil {
		return types.Claim{}, false
	}

	k.cdc.MustUnmarshal(claimBz, &claim)

	return claim, true
}
