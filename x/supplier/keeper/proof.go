package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// UpsertProof inserts or updates a specific proof in the store by index.
func (k Keeper) UpsertProof(ctx sdk.Context, proof types.Proof) {
	logger := k.Logger(ctx).With("method", "UpsertProof")

	proofBz := k.cdc.MustMarshal(&proof)
	parentStore := ctx.KVStore(k.storeKey)

	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	sessionId := proof.GetSessionHeader().GetSessionId()
	primaryKey := types.ProofPrimaryKey(sessionId, proof.GetSupplierAddress())
	primaryStore.Set(primaryKey, proofBz)

	logger.Info(fmt.Sprintf("upserted proof for supplier %s with primaryKey %s", proof.GetSupplierAddress(), primaryKey))

	// Update the address index: supplierAddress -> [ProofPrimaryKey]
	addressStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ProofSupplierAddressPrefix))
	addressKey := types.ProofSupplierAddressKey(proof.GetSupplierAddress(), primaryKey)
	addressStoreIndex.Set(addressKey, primaryKey)

	logger.Info(fmt.Sprintf("indexed Proof for supplier %s with primaryKey %s", proof.GetSupplierAddress(), primaryKey))

	// Update the session end height index: sessionEndHeight -> [ProofPrimaryKey]
	sessionHeightStoreIndex := prefix.NewStore(parentStore, types.KeyPrefix(types.ProofSessionEndHeightPrefix))
	sessionEndHeight := proof.GetSessionHeader().GetSessionEndBlockHeight()
	heightKey := types.ProofSupplierEndSessionHeightKey(sessionEndHeight, primaryKey)
	sessionHeightStoreIndex.Set(heightKey, primaryKey)
}

// GetProof returns a proof from its index
func (k Keeper) GetProof(ctx sdk.Context, sessionId, supplierAdd string) (val types.Proof, found bool) {
	primaryKey := types.ProofPrimaryKey(sessionId, supplierAdd)
	return k.getProofByPrimaryKey(ctx, primaryKey)
}

// RemoveProof removes a proof from the store
func (k Keeper) RemoveProof(ctx sdk.Context, sessionId, supplierAddr string) {
	parentStore := ctx.KVStore(k.storeKey)
	proofPrimaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	proofPrimaryKey := types.ProofPrimaryKey(sessionId, supplierAddr)
	proofPrimaryStore.Delete(proofPrimaryKey)
}

// GetAllProofs returns all proof
func (k Keeper) GetAllProofs(ctx sdk.Context) (list []types.Proof) {
	parentStore := ctx.KVStore(k.storeKey)
	primaryStore := prefix.NewStore(parentStore, types.KeyPrefix(types.ProofPrimaryKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(primaryStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Proof
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// getProofByPrimaryKey is a helper that retrieves, if exists, the Proof associated with the key provided
func (k Keeper) getProofByPrimaryKey(ctx sdk.Context, primaryKey []byte) (val types.Proof, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ProofPrimaryKeyPrefix))

	proofBz := store.Get(primaryKey)
	if proofBz == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(proofBz, &val)
	return val, true
}
