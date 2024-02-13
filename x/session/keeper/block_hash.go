package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/runtime"
)

// GetBlockHash returns the hash of the block at the given height.
func (k Keeper) GetBlockHash(ctx context.Context, height int64) []byte {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return store.Get(GetBlockHashKey(height))
}

// GetBlockHashKey returns the key used to store the block hash for a given height.
func GetBlockHashKey(height int64) []byte {
	return []byte(fmt.Sprintf("Blockhash:%d", height))
}
