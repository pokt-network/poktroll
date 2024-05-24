package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// UpsertRelayMiningDifficulty set a specific relay mining difficulty in the store.
func (k Keeper) UpsertRelayMiningDifficulty(
	ctx context.Context,
	difficulty types.RelayMiningDifficulty,
) {
	logger := k.Logger().With("method", "UpsertRelayMiningDifficulty")

	difficultyBz := k.cdc.MustMarshal(&difficulty)

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))

	primaryKey := types.RelayMiningDifficultyKey(difficulty.ServiceId)
	primaryStore.Set(primaryKey, difficultyBz)

	logger.Info(
		fmt.Sprintf("upserted relay mining difficulty for service %s at height %d", difficulty.ServiceId, difficulty.BlockHeight),
	)
}

// GetClaim returns a claim from its index
func (k Keeper) GetRelayMiningDifficulty(
	ctx context.Context, serviceId string,
) (
	difficulty types.RelayMiningDifficulty, isFound bool,
) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))

	primaryKey := types.RelayMiningDifficultyKey(serviceId)
	difficultyBz := primaryStore.Get(primaryKey)

	if difficultyBz == nil {
		return types.RelayMiningDifficulty{}, false
	}

	k.cdc.MustUnmarshal(difficultyBz, &difficulty)

	return difficulty, true
}
