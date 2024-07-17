package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SetRelayMiningDifficulty set a specific relayMiningDifficulty in the store from its index
func (k Keeper) SetRelayMiningDifficulty(ctx context.Context, relayMiningDifficulty tokenomics.RelayMiningDifficulty) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	difficultyBz := k.cdc.MustMarshal(&relayMiningDifficulty)
	store.Set(types.RelayMiningDifficultyKey(
		relayMiningDifficulty.ServiceId,
	), difficultyBz)
}

// GetRelayMiningDifficulty returns a relayMiningDifficulty from its index
func (k Keeper) GetRelayMiningDifficulty(
	ctx context.Context,
	serviceId string,
) (difficulty tokenomics.RelayMiningDifficulty, found bool) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))

	difficultyBz := store.Get(types.RelayMiningDifficultyKey(
		serviceId,
	))
	if difficultyBz == nil {
		difficulty.ServiceId = serviceId
		return difficulty, false
	}

	k.cdc.MustUnmarshal(difficultyBz, &difficulty)
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
}

// GetAllRelayMiningDifficulty returns all relayMiningDifficulty
func (k Keeper) GetAllRelayMiningDifficulty(ctx context.Context) (list []tokenomics.RelayMiningDifficulty) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val tokenomics.RelayMiningDifficulty
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
