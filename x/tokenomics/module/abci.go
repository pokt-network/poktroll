package tokenomics

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// EndBlocker called at every block and settles all pending claims.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) (err error) {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	logger := k.Logger().With("method", "EndBlocker")

	// DEV_NOTE: There are two primary reasons why claims are settled at the EndBlocker instead of proof submission:
	// 1. Logic - Probabilistic proof allows claims to be settled (i.e. rewarded)
	//    even without a proof to be able to scale to unbounded Claims & Proofs.
	// 2. Implementation - This cannot be done from the `x/proof` module because
	//    it would create a circular dependency.
	settledResults, expiredResults, numDiscardedFaultyClaims, err := k.SettlePendingClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not settle pending claims due to error %v", err))
		return err
	}
	logger.Info(fmt.Sprintf(
		"settled %d claims, expired %d claims, discarded %d faulty claims",
		settledResults.GetNumClaims(),
		expiredResults.GetNumClaims(),
		numDiscardedFaultyClaims,
	))
	// Secondary warning log to alert of non-zero discarded faulty claims.
	if numDiscardedFaultyClaims > 0 {
		logger.Warn(fmt.Sprintf("discarded %d faulty claims", numDiscardedFaultyClaims))
	}

	// Update the relay mining difficulty for every service that settled pending claims.
	settledRelaysPerServiceIdMap, err := settledResults.GetRelaysPerServiceMap()
	if err != nil {
		logger.Error(fmt.Sprintf("could not get settledRelaysPerServiceIdMap due to error: %v", err))
		return err
	}
	difficultyPerServiceMap, err := k.UpdateRelayMiningDifficulty(ctx, settledRelaysPerServiceIdMap)
	if err != nil {
		logger.Error(fmt.Sprintf("could not update relay mining difficulties due to error: %v", err))
		return err
	}
	logger.Info(fmt.Sprintf(
		"successfully updated relay mining difficulties for %d services",
		len(difficultyPerServiceMap),
	))

	// Telemetry - emit telemetry for each service's relay mining difficulty.
	for serviceId, newRelayMiningDifficulty := range difficultyPerServiceMap {
		// DEV_NOTE: The difficulty integer is a human readable interpretation of
		// the target hash intended for telemetry purposes only.
		difficulty := protocol.GetRelayDifficultyMultiplierToFloat32(newRelayMiningDifficulty.TargetHash)
		telemetry.RelayMiningDifficultyGauge(difficulty, serviceId)
		telemetry.RelayEMAGauge(newRelayMiningDifficulty.NumRelaysEma, serviceId)
		logger.Debug(fmt.Sprintf(
			"Updated relay mining difficulty for service %q with difficulty %f and EMA %d",
			serviceId,
			difficulty,
			newRelayMiningDifficulty.NumRelaysEma,
		))
	}

	return nil
}
