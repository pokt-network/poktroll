package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TargetNumRelays is the target number of relays we want the network to mine for
// a specific service across all applications & suppliers per session.
// This number determines the total number of leafs to be created across in
// the off-chain SMTs, across all suppliers, for each service.
// It indirectly drives the off-chain resource requirements of the network
// in additional to playing a critical role in Relay Mining.
// TODO_BLOCKER(@Olshansk, #542): Make this a governance parameter.
const TargetNumRelays = uint64(10e4)

// Exponential moving average (ema) smoothing factor, commonly known as alpha.
// Usually, alpha = 2 / (N+1), where N is the number of periods.
// Large alpha -> more weight on recent data; less smoothing and fast response.
// Small alpha -> more weight on past data; more smoothing and slow response.
//
// TODO_MAINNET: Use a language agnostic float implementation or arithmetic library
// to ensure deterministic results across different language implementations of the
// protocol.
var emaSmoothingFactor = new(big.Float).SetFloat64(0.1)

// UpdateRelayMiningDifficulty updates the on-chain relay mining difficulty
// based on the amount of on-chain relays for each service, given a map of serviceId->numRelays.
func (k Keeper) UpdateRelayMiningDifficulty(
	ctx context.Context,
	relaysPerServiceMap map[string]uint64,
) (difficultyPerServiceMap map[string]types.RelayMiningDifficulty, err error) {
	logger := k.Logger().With("method", "UpdateRelayMiningDifficulty")
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	difficultyPerServiceMap = make(map[string]types.RelayMiningDifficulty, len(relaysPerServiceMap))
	for serviceId, numRelays := range relaysPerServiceMap {
		prevDifficulty, found := k.GetRelayMiningDifficulty(ctx, serviceId)
		if !found {
			logger.Warn(types.ErrTokenomicsMissingRelayMiningDifficulty.Wrapf(
				"No previous relay mining difficulty found for service %s. Initializing with default difficulty %v",
				serviceId, prevDifficulty.TargetHash,
			).Error())

			// If a previous difficulty for the service is not found, we initialize
			// it with a default.
			prevDifficulty = types.RelayMiningDifficulty{
				ServiceId:    serviceId,
				BlockHeight:  sdkCtx.BlockHeight(),
				NumRelaysEma: numRelays,
				TargetHash:   prooftypes.DefaultRelayDifficultyTargetHash,
			}
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
		difficultyHash := ComputeNewDifficultyTargetHash(prevDifficulty.TargetHash, TargetNumRelays, newRelaysEma)
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

// ComputeNewDifficultyTargetHash computes the new difficulty target hash based
// on the target number of relays we want the network to mine and the new EMA of
// the number of relays.
// NB: Exported for testing purposes only.
func ComputeNewDifficultyTargetHash(prevTargetHash []byte, targetNumRelays, newRelaysEma uint64) []byte {
	// The target number of relays we want the network to mine is greater than
	// the actual on-chain relays, so we don't need to scale to anything above
	// the default.
	if targetNumRelays > newRelaysEma {
		return prooftypes.DefaultRelayDifficultyTargetHash
	}

	// Calculate the proportion of target relays to the new EMA
	// TODO_MAINNET: Use a language agnostic float implementation or arithmetic library
	// to ensure deterministic results across different language implementations of the
	// protocol.
	ratio := new(big.Float).Quo(
		new(big.Float).SetUint64(targetNumRelays),
		new(big.Float).SetUint64(newRelaysEma),
	)

	// Compute the new target hash by scaling the previous target hash based on the ratio
	newTargetHash := scaleDifficultyTargetHash(prevTargetHash, ratio)

	return newTargetHash
}

// scaleDifficultyTargetHash scales the target hash based on the given ratio
//
// TODO_MAINNET: Use a language agnostic float implementation or arithmetic library
// to ensure deterministic results across different language implementations of the
// protocol.
func scaleDifficultyTargetHash(targetHash []byte, ratio *big.Float) []byte {
	// Convert targetHash to a big.Float to miminize precision loss.
	targetInt := new(big.Int).SetBytes(targetHash)
	targetFloat := new(big.Float).SetInt(targetInt)

	// Scale the target by multiplying it by the ratio.
	scaledTargetFloat := new(big.Float).Mul(targetFloat, ratio)
	// NB: Some precision is lost when converting back to an integer.
	scaledTargetInt, _ := scaledTargetFloat.Int(nil)
	scaledTargetHash := scaledTargetInt.Bytes()

	// Ensure the scaled target hash maxes out at Difficulty1.
	if len(scaledTargetHash) > len(targetHash) {
		return protocol.Difficulty1HashBz
	}

	// Ensure the scaled target hash has the same length as the default target hash.
	if len(scaledTargetHash) < len(targetHash) {
		paddedTargetHash := make([]byte, len(targetHash))
		copy(paddedTargetHash[len(paddedTargetHash)-len(scaledTargetHash):], scaledTargetHash)
		return paddedTargetHash
	}

	return scaledTargetHash
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
