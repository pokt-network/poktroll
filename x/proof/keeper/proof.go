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
	primaryKey := types.ProofPrimaryKey(sessionId, proof.GetSupplierAddress())
	primaryStore.Set(primaryKey, proofBz)

	logger.Info(fmt.Sprintf("upserted proof for supplier %s with primaryKey %s", proof.GetSupplierAddress(), primaryKey))

	// Update the address index: supplierAddress -> [ProofPrimaryKey]
	addressStoreIndex := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofSupplierAddressPrefix))
	addressKey := types.ProofSupplierAddressKey(proof.GetSupplierAddress(), primaryKey)
	addressStoreIndex.Set(addressKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed Proof for supplier %s with primaryKey %s", proof.GetSupplierAddress(), primaryKey))

	// Update the session end height index: sessionEndHeight -> [ProofPrimaryKey]
	sessionHeightStoreIndex := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofSessionEndHeightPrefix))
	sessionEndHeight := proof.GetSessionHeader().GetSessionEndBlockHeight()
	heightKey := types.ProofSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)
	sessionHeightStoreIndex.Set(heightKey, primaryKey)
}

// GetProof returns a proof from its index
func (k Keeper) GetProof(ctx context.Context, sessionId, supplierAdd string) (val types.Proof, found bool) {
	primaryKey := types.ProofPrimaryKey(sessionId, supplierAdd)
	return k.getProofByPrimaryKey(ctx, primaryKey)
}

// RemoveProof removes a proof from the store
func (k Keeper) RemoveProof(ctx context.Context, sessionId, supplierAddr string) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	primaryKey := types.ProofPrimaryKey(sessionId, supplierAddr)
	primaryStore.Delete(primaryKey)
}

// GetAllProofs returns all proof
func (k Keeper) GetAllProofs(ctx context.Context) (list []types.Proof) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(primaryStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Proof
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// getProofByPrimaryKey is a helper that retrieves, if exists, the Proof associated with the key provided
func (k Keeper) getProofByPrimaryKey(ctx context.Context, primaryKey []byte) (proof types.Proof, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.ProofPrimaryKeyPrefix))

	proofBz := primaryStore.Get(primaryKey)
	if proofBz == nil {
		return proof, false
	}

	k.cdc.MustUnmarshal(proofBz, &proof)
	return proof, true
}
