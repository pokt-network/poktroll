package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
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
}

// GetRelayMiningDifficulty returns a relayMiningDifficulty from its index
func (k Keeper) GetRelayMiningDifficulty(
	ctx context.Context,
	serviceId string,
) (difficulty types.RelayMiningDifficulty, found bool) {
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
func (k Keeper) GetAllRelayMiningDifficulty(ctx context.Context) (list []types.RelayMiningDifficulty) {
	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.RelayMiningDifficultyKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.RelayMiningDifficulty
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// SetRelayMiningDifficultyAtHeight stores a snapshot of relay mining difficulty
// with its effective height for historical lookups.
func (k Keeper) SetRelayMiningDifficultyAtHeight(
	ctx context.Context,
	effectiveHeight int64,
	difficulty types.RelayMiningDifficulty,
) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	difficultyUpdate := types.RelayMiningDifficultyUpdate{
		EffectiveHeight: effectiveHeight,
		Difficulty:      &difficulty,
	}

	bz, err := k.cdc.Marshal(&difficultyUpdate)
	if err != nil {
		return err
	}

	key := types.RelayMiningDifficultyHistoryKey(difficulty.ServiceId, effectiveHeight)
	store.Set(key, bz)

	return nil
}

// GetRelayMiningDifficultyAtHeight returns the relay mining difficulty that was
// effective at the given height for a specific service.
// It finds the most recent difficulty entry where effective_height <= queryHeight.
// If no historical difficulty exists, it returns the current difficulty (backwards compatible).
func (k Keeper) GetRelayMiningDifficultyAtHeight(
	ctx context.Context,
	serviceId string,
	queryHeight int64,
) (types.RelayMiningDifficulty, bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	// Create prefix store for this service's history
	serviceHistoryPrefix := types.RelayMiningDifficultyHistoryKeyPrefixForService(serviceId)
	historyStore := prefix.NewStore(store, serviceHistoryPrefix)

	// Create end key for reverse iteration (exclusive upper bound)
	endKey := make([]byte, 8)
	binary.BigEndian.PutUint64(endKey, uint64(queryHeight+1))

	// Reverse iterate to find the most recent entry <= queryHeight
	iterator := historyStore.ReverseIterator(nil, endKey)
	defer iterator.Close()

	if iterator.Valid() {
		var difficultyUpdate types.RelayMiningDifficultyUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &difficultyUpdate)
		if difficultyUpdate.Difficulty != nil {
			return *difficultyUpdate.Difficulty, true
		}
	}

	// Fallback: If no historical difficulty found, return a deterministic base
	// difficulty. This avoids non-determinism from node-local store state when
	// no history exists (e.g. for services created after an upgrade handler).
	targetNumRelays := k.GetParams(ctx).TargetNumRelays
	return types.RelayMiningDifficulty{
		ServiceId:    serviceId,
		BlockHeight:  0,
		NumRelaysEma: targetNumRelays,
		TargetHash:   protocol.BaseRelayDifficultyHashBz,
	}, false
}

// GetAllRelayMiningDifficultyHistory returns all historical difficulty updates.
// This is primarily used for debugging and testing.
func (k Keeper) GetAllRelayMiningDifficultyHistory(ctx context.Context) []types.RelayMiningDifficultyUpdate {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	historyStore := prefix.NewStore(store, []byte(types.RelayMiningDifficultyHistoryKeyPrefix))

	iterator := historyStore.Iterator(nil, nil)
	defer iterator.Close()

	var history []types.RelayMiningDifficultyUpdate
	for ; iterator.Valid(); iterator.Next() {
		var difficultyUpdate types.RelayMiningDifficultyUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &difficultyUpdate)
		history = append(history, difficultyUpdate)
	}

	return history
}

// GetRelayMiningDifficultyHistoryForService returns all historical difficulty
// updates for a specific service.
func (k Keeper) GetRelayMiningDifficultyHistoryForService(
	ctx context.Context,
	serviceId string,
) []types.RelayMiningDifficultyUpdate {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceHistoryPrefix := types.RelayMiningDifficultyHistoryKeyPrefixForService(serviceId)
	historyStore := prefix.NewStore(store, serviceHistoryPrefix)

	iterator := historyStore.Iterator(nil, nil)
	defer iterator.Close()

	var history []types.RelayMiningDifficultyUpdate
	for ; iterator.Valid(); iterator.Next() {
		var difficultyUpdate types.RelayMiningDifficultyUpdate
		k.cdc.MustUnmarshal(iterator.Value(), &difficultyUpdate)
		history = append(history, difficultyUpdate)
	}

	return history
}
