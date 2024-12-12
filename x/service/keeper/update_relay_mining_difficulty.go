package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/x/service/types"
)

var (
	// Exponential moving average (ema) smoothing factor, commonly known as alpha.
	// Usually, alpha = 2 / (N+1), where N is the number of periods.
	// Large alpha -> more weight on recent data; less smoothing and fast response.
	// Small alpha -> more weight on past data; more smoothing and slow response.
	//
	// TODO_MAINNET: Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	//
	// TODO_MAINNET(@olshansk, @rawthil): Play around with the value N for EMA to
	// capture what the memory should be.
	emaSmoothingFactor = new(big.Float).SetFloat64(0.1)
)

// UpdateRelayMiningDifficulty updates the on-chain relay mining difficulty
// based on the amount of on-chain relays for each service, given a map of serviceId->numRelays.
func (k Keeper) UpdateRelayMiningDifficulty(
	ctx context.Context,
	relaysPerServiceMap map[string]uint64,
) (difficultyPerServiceMap map[string]types.RelayMiningDifficulty, err error) {
	logger := k.Logger().With("method", "UpdateRelayMiningDifficulty")
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	difficultyPerServiceMap = make(map[string]types.RelayMiningDifficulty, len(relaysPerServiceMap))

	// Iterate over the relaysPerServiceMap deterministically by sorting the keys.
	// This ensures that the order of the keys is consistent across different nodes.
	// See comment: https://github.com/pokt-network/poktroll/pull/840#discussion_r1796663285
	targetNumRelays := k.GetParams(ctx).TargetNumRelays
	sortedRelayPerServiceMapKeys := getSortedMapKeys(relaysPerServiceMap)
	for _, serviceId := range sortedRelayPerServiceMapKeys {
		numRelays := relaysPerServiceMap[serviceId]
		prevDifficulty, found := k.GetRelayMiningDifficulty(ctx, serviceId)
		if !found {
			prevDifficulty = NewDefaultRelayMiningDifficulty(
				ctx,
				logger,
				serviceId,
				numRelays,
				targetNumRelays,
			)
		}

		// TODO_MAINNET(@Olshansk): We could potentially compute the smoothing factor
		// using a common formula, such as alpha = 2 / (N+1), where N is the number
		// of periods.
		// N := ctx.BlockHeight() - prevDifficulty.BlockHeight
		// alpha := 2 / (1 + N)
		alpha := emaSmoothingFactor

		// Compute the updated EMA of the number of relays.
		prevRelaysEma := prevDifficulty.NumRelaysEma
		newRelaysEma := computeEma(alpha, prevRelaysEma, numRelays)

		// CRITICAL_DEV_NOTE: We changed this code to pass in  "BaseRelayDifficultyHashBz" instead of "prevDifficulty.TargetHash"
		// to "ComputeNewDifficultyTargetHash" because we used to have 2 moving variables:
		// 		1. Input difficulty
		// 		2. Relays EMA
		// However, since the "TargetNumRelays" remained constant, the following case would keep scaling down the difficulty:
		// 		- newRelaysEma = 100 -> scaled by 10 / 100 -> scaled down by 0.1
		// 		- newRelaysEma = 50 -> scaled by 10 / 50 -> scaled down by 0.2
		// 		- newRelaysEma = 20 -> scaled by 10 / 20 -> scaled down by 0.5
		// We kept scaling down even though numRelaysEma was decreasing.
		// To avoid continuing to increase the difficulty (i.e. scaling down), the
		// relative starting difficulty has to be kept constant.
		difficultyHash := protocol.ComputeNewDifficultyTargetHash(protocol.BaseRelayDifficultyHashBz, targetNumRelays, newRelaysEma)
		newDifficulty := types.RelayMiningDifficulty{
			ServiceId:    serviceId,
			BlockHeight:  sdkCtx.BlockHeight(),
			NumRelaysEma: newRelaysEma,
			TargetHash:   difficultyHash,
		}
		k.SetRelayMiningDifficulty(ctx, newDifficulty)

		// Emit an event for the updated relay mining difficulty regardless of
		// whether the difficulty changed or not.

		relayMiningDifficultyUpdateEvent := types.EventRelayMiningDifficultyUpdated{
			ServiceId:                serviceId,
			PrevTargetHashHexEncoded: hex.EncodeToString(prevDifficulty.TargetHash),
			NewTargetHashHexEncoded:  hex.EncodeToString(newDifficulty.TargetHash),
			PrevNumRelaysEma:         prevDifficulty.NumRelaysEma,
			NewNumRelaysEma:          newDifficulty.NumRelaysEma,
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(&relayMiningDifficultyUpdateEvent); err != nil {
			return nil, err
		}

		// Output the appropriate log message based on whether the difficulty was initialized, updated or unchanged.
		var logMessage string
		switch {
		case !found:
			logMessage = fmt.Sprintf("Initialized RelayMiningDifficulty for service %s at height %d with difficulty %x", serviceId, sdkCtx.BlockHeight(), newDifficulty.TargetHash)
		case !bytes.Equal(prevDifficulty.TargetHash, newDifficulty.TargetHash):
			logMessage = fmt.Sprintf("Updated RelayMiningDifficulty for service %s at height %d from %x to %x", serviceId, sdkCtx.BlockHeight(), prevDifficulty.TargetHash, newDifficulty.TargetHash)
		default:
			logMessage = fmt.Sprintf("No change in RelayMiningDifficulty for service %s at height %d. Current difficulty: %x", serviceId, sdkCtx.BlockHeight(), newDifficulty.TargetHash)
		}
		logger.Info(logMessage)

		// Store the updated difficulty in the map for telemetry.
		// This is done to only emit the telemetry event if all the difficulties
		// are updated successfully.
		difficultyPerServiceMap[serviceId] = newDifficulty
	}

	return difficultyPerServiceMap, nil
}

// computeEma computes the EMA at time t, given the EMA at time t-1, the raw
// data revealed at time t, and the smoothing factor α.
// Src: https://en.wikipedia.org/wiki/Exponential_smoothing
//
// TODO_MAINNET: Use a language agnostic float implementation or arithmetic library
// to ensure deterministic results across different language implementations of the
// protocol.
func computeEma(alpha *big.Float, prevEma, currValue uint64) uint64 {
	oneMinusAlpha := new(big.Float).Sub(new(big.Float).SetInt64(1), alpha)
	prevEmaFloat := new(big.Float).SetUint64(prevEma)

	weightedCurrentContribution := new(big.Float).Mul(alpha, new(big.Float).SetUint64(currValue))
	weightedPreviousContribution := new(big.Float).Mul(oneMinusAlpha, prevEmaFloat)
	newEma, _ := new(big.Float).Add(weightedCurrentContribution, weightedPreviousContribution).Uint64()
	return newEma
}

// NewDefaultRelayMiningDifficulty is a helper that creates a new RelayMiningDifficulty
// structure if one is not available. It is often used to set the default when a service's
// difficulty is being initialized for the first time.
func NewDefaultRelayMiningDifficulty(
	ctx context.Context,
	logger log.Logger,
	serviceId string,
	numRelays uint64,
	targetNumRelays uint64,
) types.RelayMiningDifficulty {
	logger = logger.With("helper", "NewDefaultRelayMiningDifficulty")

	// Compute the target hash based on the number of relays seen for the first time.
	newDifficultyHash := protocol.ComputeNewDifficultyTargetHash(protocol.BaseRelayDifficultyHashBz, targetNumRelays, numRelays)

	logger.Warn(types.ErrServiceMissingRelayMiningDifficulty.Wrapf(
		"No previous relay mining difficulty found for service %s.\n"+
			"Creating a new relay mining difficulty with %d relays and an initial target hash %x",
		serviceId, numRelays, newDifficultyHash).Error())

	// Return a new RelayMiningDifficulty with the computed target hash.
	return types.RelayMiningDifficulty{
		ServiceId:    serviceId,
		BlockHeight:  sdk.UnwrapSDKContext(ctx).BlockHeight(),
		NumRelaysEma: numRelays,
		TargetHash:   newDifficultyHash,
	}

}

// getSortedMapKeys returns the keys of a map lexicographically sorted.
func getSortedMapKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}
