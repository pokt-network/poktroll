package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// UpsertClaim set a specific claim in the store from its index
func (k Keeper) UpsertClaim(ctx context.Context, claim proof.Claim) {
	logger := k.Logger().With("method", "UpsertClaim")

	claimBz := k.cdc.MustMarshal(&claim)
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	sessionId := claim.GetSessionHeader().GetSessionId()
	primaryKey := types.ClaimPrimaryKey(sessionId, claim.SupplierAddress)
	primaryStore.Set(primaryKey, claimBz)
	logger.Info(fmt.Sprintf("upserted claim for supplier %s with primaryKey %s", claim.SupplierAddress, primaryKey))

	// Update the address index: supplierAddress -> [ClaimPrimaryKey]
	supplierAddrStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSupplierAddressPrefix))
	supplierAddrKey := types.ClaimSupplierAddressKey(claim.SupplierAddress, primaryKey)
	supplierAddrStore.Set(supplierAddrKey, primaryKey)
	logger.Info(fmt.Sprintf("indexed claim for supplier %s with primaryKey %s", claim.SupplierAddress, primaryKey))

	// Update the session end height index: sessionEndHeight -> [ClaimPrimaryKey]
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))
	sessionEndHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	sessionEndHeightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)
	sessionEndHeightStore.Set(sessionEndHeightKey, primaryKey)
	logger.Info(fmt.Sprintf("indexed claim for supplier %s at session ending height %d", claim.SupplierAddress, sessionEndHeight))
}

// GetClaim returns a claim from its index
func (k Keeper) GetClaim(ctx context.Context, sessionId, supplierAddr string) (_ proof.Claim, isClaimFound bool) {
	return k.getClaimByPrimaryKey(ctx, types.ClaimPrimaryKey(sessionId, supplierAddr))
}

// RemoveClaim removes a claim from the store
func (k Keeper) RemoveClaim(ctx context.Context, sessionId, supplierAddr string) {
	logger := k.Logger().With("method", "RemoveClaim")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))

	// Check if the claim exists
	primaryKey := types.ClaimPrimaryKey(sessionId, supplierAddr)
	foundClaim, isClaimFound := k.getClaimByPrimaryKey(ctx, primaryKey)
	if !isClaimFound {
		logger.Error(fmt.Sprintf("trying to delete non-existent claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId))
		return
	}

	// Prepare the indices for deletion
	supplierAddrStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSupplierAddressPrefix))
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimSessionEndHeightPrefix))

	supplierAddrKey := types.ClaimSupplierAddressKey(foundClaim.GetSupplierAddress(), primaryKey)
	sessionEndHeight := foundClaim.GetSessionHeader().GetSessionEndBlockHeight()
	sessionEndHeightKey := types.ClaimSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)

	// Delete all the entries (primary store and secondary indices)
	primaryStore.Delete(primaryKey)
	supplierAddrStore.Delete(supplierAddrKey)
	sessionEndHeightStore.Delete(sessionEndHeightKey)

	logger.Info(fmt.Sprintf("deleted claim with primary key %s for supplier %s and session %s", primaryKey, supplierAddr, sessionId))
}

// GetAllClaims returns all claim
func (k Keeper) GetAllClaims(ctx context.Context) (claims []proof.Claim) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(primaryStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var claim proof.Claim
		k.cdc.MustUnmarshal(iterator.Value(), &claim)
		claims = append(claims, claim)
	}

	return claims
}

// getClaimByPrimaryKey is a helper that retrieves, if exists, the Claim associated with the key provided
func (k Keeper) getClaimByPrimaryKey(ctx context.Context, primaryKey []byte) (claim proof.Claim, isClaimFound bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ClaimPrimaryKeyPrefix))
	claimBz := primaryStore.Get(primaryKey)

	if claimBz == nil {
		return proof.Claim{}, false
	}

	k.cdc.MustUnmarshal(claimBz, &claim)

	return claim, true
}
