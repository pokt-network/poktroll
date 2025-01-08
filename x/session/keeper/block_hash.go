package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/session/types"
)

// GetBlockHash returns the hash of the block at the given height.
func (k Keeper) GetBlockHash(ctx context.Context, height int64) []byte {
	if k.blockHashes[height] != nil {
		return k.blockHashes[height]
	}
	// There is no block hash stored for the genesis block (height 0),
	// in this case return an empty byte slice.
	if height <= 0 {
		return []byte{}
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.BlockHashKeyPrefix))
	blockHash := store.Get(types.BlockHashKey(height))
	k.blockHashes[height] = blockHash
	return blockHash
}

func (k Keeper) ResetCache() {
	k.cachedParams = nil
	clear(k.blockHashes)
}
