package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/session/types"
)

// GetBlockHash returns the hash of the block at the given height.
func (k Keeper) GetBlockHash(ctx context.Context, height int64) []byte {
	// If the block is not a consensus produced block, then we don't have the hash,
	// so we return an empty byte slice.
	if height <= 0 {
		return []byte{}
	}

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.BlockHashKeyPrefix))
	return store.Get(types.BlockHashKey(height))
}
