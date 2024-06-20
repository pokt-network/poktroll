package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_UPNET(@Olshansk, #542): Add telemetry that will enable:
// 1. Visualizing a multi-line chart of "Relays EMA per Service" (title) of "Relay EMA" (y-axis) vs block/time (x-axis) and being able to select each service.
// 1. Visualizing a multi-line chart of "Relay Mining Difficulty per service" (title) of "Relay EMA" (y-axis) vs block/time (x-axis) and being able to select each service.

const (
	// Exponential moving average (ema) smoothing factor, commonly known as alpha.
	// Usually, alpha = 2 / (N+1), where N is the number of periods.
	// Large alpha -> more weight on recent data; less smoothing and fast response.
	// Small alpha -> more weight on past data; more smoothing and slow response.
	emaSmoothingFactor = float64(0.1)

	// The target number of relays we want the network to mine for a specific
	// service across all applications & suppliers per session.
	// This number determines the total number of leafs to be created across in
	// the off-chain SMTs, across all suppliers, for each service.
	// It indirectly drives the off-chain resource requirements of the network
	// in additional to playing a critical role in Relay Mining.
	// TODO_BLOCKER(@Olshansk, #542): Make this a governance parameter.
	TargetNumRelays = uint64(10e4)
)

// UpdateRelayMiningDifficulty updates the on-chain relay mining difficulty
// based on the amount of on-chain relays for each service, given a map of serviceId->numRelays.
func (k Keeper) UpdateRelayMiningDifficulty(
	ctx context.Context,
	relaysPerServiceMap map[string]uint64,
) error {
	logger := k.Logger().With("method", "UpdateRelayMiningDifficulty")
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for serviceId, numRelays := range relaysPerServiceMap {
		prevDifficulty, found := k.GetRelayMiningDifficulty(ctx, serviceId)
		if !found {
			types.ErrTokenomicsMissingRelayMiningDifficulty.Wrapf("No previous relay mining difficulty found for service %s. Initializing with default difficulty %v", serviceId, prevDifficulty.TargetHash)
			// If a previous difficulty for the service is not found, we initialize
			// it with a default.
			prevDifficulty = types.RelayMiningDifficulty{
				ServiceId:    serviceId,
				BlockHeight:  sdkCtx.BlockHeight(),
				NumRelaysEma: numRelays,
				TargetHash:   defaultDifficultyTargetHash(),
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
		difficultyHash := ComputeNewDifficultyTargetHash(TargetNumRelays, newRelaysEma)
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
			return err
		}

		// Output the appropriate log message based on whether the difficulty was initialized, updated or unchanged.
		var logMessage string
		if !found {
			logMessage = fmt.Sprintf("Initialized RelayMiningDifficulty for service %s at height %d with difficulty %x", serviceId, sdkCtx.BlockHeight(), newDifficulty.TargetHash)
		} else if !bytes.Equal(prevDifficulty.TargetHash, newDifficulty.TargetHash) {
			logMessage = fmt.Sprintf("Updated RelayMiningDifficulty for service %s at height %d from %x to %x", serviceId, sdkCtx.BlockHeight(), prevDifficulty.TargetHash, newDifficulty.TargetHash)
		} else {
			logMessage = fmt.Sprintf("No change in RelayMiningDifficulty for service %s at height %d. Current difficulty: %x", serviceId, sdkCtx.BlockHeight(), newDifficulty.TargetHash)
		}
		logger.Info(logMessage)
	}

	return nil
}

// ComputeNewDifficultyTargetHash computes the new difficulty target hash based
// on the target number of relays we want the network to mine and the new EMA of
// the number of relays.
// NB: Exported for testing purposes only.
func ComputeNewDifficultyTargetHash(targetNumRelays, newRelaysEma uint64) []byte {
	// The target number of relays we want the network to mine is greater than
	// the actual on-chain relays, so we don't need to scale to anything above
	// the default.
	if targetNumRelays > newRelaysEma {
		return defaultDifficultyTargetHash()
	}

	log2 := func(x float64) float64 {
		return math.Log(x) / math.Ln2
	}

	// We are dealing with a bitwise binary distribution, and are trying to convert
	// the proportion of an off-chain relay (i.e. relayEMA) to an
	// on-chain relay (i.e. target) based on the probability of x leading zeros
	// in the target hash.
	//
	// In other words, the probability of an off-chain relay moving into the tree
	// should equal (approximately) the probability of having x leading zeroes
	// in the target hash.
	//
	// The construction is as follows:
	// (0.5)^num_leading_zeroes = (num_target_relay / num_total_relays)
	// (0.5)^x = (T/R)
	// 	x = -ln2(T/R)
	numLeadingZeroBits := int(-log2(float64(targetNumRelays) / float64(newRelaysEma)))
	numBytes := proofkeeper.SmtSpec.PathHasherSize()
	return LeadingZeroBitsToTargetDifficultyHash(numLeadingZeroBits, numBytes)
}

// defaultDifficultyTargetHash returns the default difficulty target hash with
// the default number of leading zero bits.
func defaultDifficultyTargetHash() []byte {
	numBytes := proofkeeper.SmtSpec.PathHasherSize()
	numDefaultLeadingZeroBits := int(prooftypes.DefaultMinRelayDifficultyBits)
	return LeadingZeroBitsToTargetDifficultyHash(numDefaultLeadingZeroBits, numBytes)
}

// computeEma computes the EMA at time t, given the EMA at time t-1, the raw
// data revealed at time t, and the smoothing factor α.
// Src: https://en.wikipedia.org/wiki/Exponential_smoothing
func computeEma(alpha float64, prevEma, currValue uint64) uint64 {
	return uint64(alpha*float64(currValue) + (1-alpha)*float64(prevEma))
}

// LeadingZeroBitsToTargetDifficultyHash generates a slice of bytes with the specified number of leading zero bits
// NB: Exported for testing purposes only.
func LeadingZeroBitsToTargetDifficultyHash(numLeadingZeroBits int, numBytes int) []byte {
	targetDifficultyHah := make([]byte, numBytes)

	// Set everything to 1s initially
	for i := range targetDifficultyHah {
		targetDifficultyHah[i] = 0xff
	}

	// Set full zero bytes
	fullZeroBytes := numLeadingZeroBits / 8
	for i := 0; i < fullZeroBytes; i++ {
		targetDifficultyHah[i] = 0
	}

	// Set remaining bits in the next byte
	remainingZeroBits := numLeadingZeroBits % 8
	if remainingZeroBits > 0 {
		targetDifficultyHah[fullZeroBytes] = byte(0xff >> remainingZeroBits)
	}

	return targetDifficultyHah
}
