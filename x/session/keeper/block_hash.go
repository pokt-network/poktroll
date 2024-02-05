package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetBlockHash returns the hash of the block at the given height.
func (k Keeper) GetBlockHash(ctx sdk.Context, height int64) []byte {
	store := ctx.KVStore(k.storeKey)
	return store.Get(GetBlockHashKey(height))
}

// GetBlockHashKey returns the key used to store the block hash for a given height.
func GetBlockHashKey(height int64) []byte {
	return []byte(fmt.Sprintf("Blockhash:%d", height))
}
