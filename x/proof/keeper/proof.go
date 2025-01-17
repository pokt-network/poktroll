package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// UpsertProof set a specific proof in the store from its index
func (k Keeper) UpsertProof(ctx context.Context, proof types.Proof) {
	logger := k.Logger().With("method", "UpsertProof")

	proofBz := k.cdc.MustMarshal(&proof)
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	sessionId := proof.GetSessionHeader().GetSessionId()
	primaryKey := types.ProofPrimaryKey(sessionId, proof.GetSupplierOperatorAddress())
	primaryStore.Set(primaryKey, proofBz)

	k.cache.Proofs[sessionId] = &proof

	logger.Info(
		fmt.Sprintf("upserted proof for supplier %s with primaryKey %s", proof.GetSupplierOperatorAddress(), primaryKey),
	)

	// Update the address index: supplierOperatorAddress -> [ProofPrimaryKey]
	supplierOperatorAddrStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofSupplierOperatorAddressPrefix))
	supplierOperatorAddrKey := types.ProofSupplierOperatorAddressKey(proof.GetSupplierOperatorAddress(), primaryKey)
	supplierOperatorAddrStore.Set(supplierOperatorAddrKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed Proof for supplier %s with primaryKey %s", proof.GetSupplierOperatorAddress(), primaryKey))

	// Update the session end height index: sessionEndHeight -> [ProofPrimaryKey]
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofSessionEndHeightPrefix))
	sessionEndHeight := proof.GetSessionHeader().GetSessionEndBlockHeight()
	sessionEndHeightKey := types.ProofSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)
	sessionEndHeightStore.Set(sessionEndHeightKey, primaryKey)
}

// GetProof returns a proof from its index
func (k Keeper) GetProof(ctx context.Context, sessionId, supplierOperatorAddr string) (_ types.Proof, isProofFound bool) {
	if proof, found := k.cache.Proofs[sessionId]; found {
		k.logger.Info("-----Proof cache hit-----")
		return *proof, true
	}

	proof, found := k.getProofByPrimaryKey(ctx, types.ProofPrimaryKey(sessionId, supplierOperatorAddr))
	if found {
		k.cache.Proofs[sessionId] = &proof
	}

	return proof, found
}

// RemoveProof removes a proof from the store
func (k Keeper) RemoveProof(ctx context.Context, sessionId, supplierOperatorAddr string) {
	logger := k.Logger().With("method", "RemoveProof")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))

	// Check if the proof exists
	primaryKey := types.ProofPrimaryKey(sessionId, supplierOperatorAddr)
	delete(k.cache.Proofs, sessionId)
	foundProof, isProofFound := k.getProofByPrimaryKey(ctx, primaryKey)
	if !isProofFound {
		logger.Error(
			fmt.Sprintf(
				"trying to delete non-existent proof with primary key %s for supplier %s and session %s",
				primaryKey,
				supplierOperatorAddr,
				sessionId,
			),
		)
		return
	}

	// Prepare the indices for deletion
	supplierOperatorAddrStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofSupplierOperatorAddressPrefix))
	sessionEndHeightStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofSessionEndHeightPrefix))

	supplierOperatorAddrKey := types.ProofSupplierOperatorAddressKey(foundProof.GetSupplierOperatorAddress(), primaryKey)
	sessionEndHeight := foundProof.GetSessionHeader().GetSessionEndBlockHeight()
	sessionEndHeightKey := types.ProofSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)

	// Delete all the entries (primary store and secondary indices)
	primaryStore.Delete(primaryKey)
	supplierOperatorAddrStore.Delete(supplierOperatorAddrKey)
	sessionEndHeightStore.Delete(sessionEndHeightKey)

	logger.Info(
		fmt.Sprintf(
			"deleted proof with primary key %s for supplier %s and session %s",
			primaryKey,
			supplierOperatorAddr,
			sessionId,
		),
	)
}

// GetAllProofsIterator returns an iterator for all proofs in the store
func (k Keeper) GetAllProofsIterator(ctx context.Context) storetypes.Iterator {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	return storetypes.KVStorePrefixIterator(primaryStore, []byte{})
}

// GetAllProofs returns all proofs in the store
func (k Keeper) GetAllProofs(ctx context.Context) (proofs []types.Proof) {
	iterator := k.GetAllProofsIterator(ctx)

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var proof types.Proof
		k.cdc.MustUnmarshal(iterator.Value(), &proof)
		k.cache.Proofs[proof.GetSessionHeader().GetSessionId()] = &proof
		proofs = append(proofs, proof)
	}

	return proofs
}

// getProofByPrimaryKey is a helper that retrieves, if exists, the Proof associated with the key provided
func (k Keeper) getProofByPrimaryKey(ctx context.Context, primaryKey []byte) (proof types.Proof, isProofFound bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))

	proofBz := primaryStore.Get(primaryKey)
	if proofBz == nil {
		return types.Proof{}, false
	}

	k.cdc.MustUnmarshal(proofBz, &proof)

	return proof, true
}
