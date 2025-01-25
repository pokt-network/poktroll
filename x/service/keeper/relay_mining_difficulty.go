package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/x/service/types"
)

// SetRelayMiningDifficulty set a specific relayMiningDifficulty in the store from its index
func (k Keeper) SetRelayMiningDifficulty(ctx context.Context, relayMiningDifficulty types.RelayMiningDifficulty) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	difficultyBz := k.cdc.MustMarshal(&relayMiningDifficulty)
	store.Set(types.RelayMiningDifficultyKey(
		relayMiningDifficulty.ServiceId,
	), difficultyBz)
	k.relayMiningDifficultyCache.Set(relayMiningDifficulty.ServiceId, relayMiningDifficulty)
}

// GetRelayMiningDifficulty returns a relayMiningDifficulty from its index
func (k Keeper) GetRelayMiningDifficulty(
	ctx context.Context,
	serviceId string,
) (difficulty types.RelayMiningDifficulty, found bool) {
	if difficulty, found := k.relayMiningDifficultyCache.Get(serviceId); found {
		k.logger.Info("-----Difficulty cache hit-----")
		return difficulty, true
	}
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))

	difficultyBz := store.Get(types.RelayMiningDifficultyKey(serviceId))
	if difficultyBz == nil {
		targetNumRelays := k.GetParams(ctx).TargetNumRelays
		k.Logger().Warn(fmt.Sprintf(
			"relayMiningDifficulty not found for service: %s, defaulting to base difficulty with protocol TargetNumRelays (%d)",
			serviceId, targetNumRelays,
		))
		difficulty = NewDefaultRelayMiningDifficulty(
			ctx,
			k.logger,
			serviceId,
			targetNumRelays,
			targetNumRelays,
		)
		return difficulty, false
	}

	k.cdc.MustUnmarshal(difficultyBz, &difficulty)
	k.relayMiningDifficultyCache.Set(serviceId, difficulty)
	return difficulty, true
}

// RemoveRelayMiningDifficulty removes a relayMiningDifficulty from the store
func (k Keeper) RemoveRelayMiningDifficulty(
	ctx context.Context,
	serviceId string,
) {
	logger := k.Logger().With("method", "RemoveRelayMiningDifficulty")

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	difficultyKey := types.RelayMiningDifficultyKey(
		serviceId,
	)

	if !store.Has(difficultyKey) {
		logger.Warn(fmt.Sprintf("trying to delete a non-existing relayMiningDifficulty for service: %s", serviceId))
		return
	}

	store.Delete(types.RelayMiningDifficultyKey(
		serviceId,
	))
	k.relayMiningDifficultyCache.Delete(serviceId)
}

// GetAllRelayMiningDifficulty returns all relayMiningDifficulty
func (k Keeper) GetAllRelayMiningDifficulty(ctx context.Context) (list []types.RelayMiningDifficulty) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var difficulty types.RelayMiningDifficulty
		k.cdc.MustUnmarshal(iterator.Value(), &difficulty)
		k.relayMiningDifficultyCache.Set(difficulty.ServiceId, difficulty)
		list = append(list, difficulty)
	}

	return
}
