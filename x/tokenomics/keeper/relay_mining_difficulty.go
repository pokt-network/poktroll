package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// UpsertClaim set a specific claim in the store from its index
func (k Keeper) UpsertRelayMiningDifficulty(
	ctx context.Context,
	serviceId string,
	difficultyTarget []byte,
	// ?? blockHeight uint64,
	// ?? blockHeight uint64,
) {
	// claimBz := k.cdc.MustMarshal(&claim)
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	primaryKey := types.RelayMiningDifficultyKey(serviceId)
	primaryStore.Set(primaryKey, difficultyTarget)
}

// GetClaim returns a claim from its index
func (k Keeper) GetRelayMiningDifficulty(
	ctx context.Context,
	serviceId string,
) (
	difficultyTarget []byte,
	isDifficultyTargetFound bool,
) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	primaryStore := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))

	primaryKey := types.RelayMiningDifficultyKey(serviceId)
	difficultyTarget = primaryStore.Get(primaryKey)

	if difficultyTarget == nil {
		return nil, false
	}
	return difficultyTarget, true
}
