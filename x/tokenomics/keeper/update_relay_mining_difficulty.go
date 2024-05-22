package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO_IN_THIS_PR: Prepare future work to decided if these should be
// constants, governance parameters or computed in some other way.
const (
	// Exponential moving average smoothing factor, commonly known as alpha.
	// Large alpha -> more weight on recent data; less smoothing and fast response.
	// Small alpha -> more weight on past data; more smoothing and slow response.
	// Usually, alpha = 2 / (N+1), where N is the number of periods.
	emaSmoothingFactor = float64(0.1)

	// The target number of compute units per service across all applications
	// and suppliers at the end of every session.
	// The target determines how to modulate the relay mining difficulty.
	targetComputeUnits = uint64(10e4)

	
)

// UpdateRelayMiningDifficulty updates the on-chain relay mining difficulty
// based on the amount of on-chain volume.
func (k Keeper) UpdateRelayMiningDifficulty(
	ctx sdk.Context,
	computeUnitsPerServiceMap map[string]uint64,
) error {
	logger := k.Logger().With("method", "UpdateRelayMiningDifficulty")

	for serviceId, computeUnits := range computeUnitsPerServiceMap {
		prevDifficultyTarget, found := k.GetRelayMiningDifficulty(serviceId)
		if !found {

		prevRelay
		newEma := computeEma(emaSmoothingFactor, revEMA, float64(computeUnits))
		if err := k.SetRelayMiningDifficulty(serviceId, newEma); err != nil {
			logger.Error(fmt.Sprintf("failed to update relay mining difficulty: %v", err))
			return err
		}
		logger.Info(fmt.Sprintf("Updated relay mining difficulty for service %s from %f to %f", serviceId, k.GetRelayMiningDifficulty(serviceId), computeUnits))
	}

	return nil
}

// computeEma computes the EMA at time t, given the EMA at time t-1, the raw
// data revealed at time t, and the smoothing factor α
// Src: https://en.wikipedia.org/wiki/Exponential_smoothing
func computeEma(alpha, prevEma, currValue float64) float64 {
	return alpha*currValue + (1-alpha)*prevEma
}

// 1: T ← 104
// ▷ Target claims by blockchain.
// 2: α ← 0.1 ▷ Exponential Moving Average Parameter.
// 3: U ← 4 ▷ Number of blocks per difficulty update.
// 4: Rema ← 0 ▷ Estimated blockchain relays, averaged by EMA.
// 5: p ← 1 ▷ Initial blockchain hash collision probability.
// 6: height ← 0
// 7: while True do
// 8: C ← getAllClaims() ▷ Get all relay claims.
// 9: R ← C
// p
// 10: Rema ← αR + (1 − α)Rema
// 11: if height%U == 0 then
// 12: p ← T
// Rema
// 13: if p > 1 then
// 14: p ← 1 ▷ If total relays are lower than target, disable relay mining.
// 15: end if
// 16: end if
// 17: height ← +1
// 18: end while
