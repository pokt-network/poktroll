package tokenomics

import (
	"context"
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// BeginBlocker called at the start of every block.
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != k.MintAllocationsSum(ctx) {
		return fmt.Errorf("mint allocation percentages do not add to 1.0")
	}

	return nil
}

// EndBlocker called at every block and settles all pending claims.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) (err error) {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	logger := k.Logger().With("method", "EndBlocker")

	// NB: There are two main reasons why we settle expiring claims in the end
	// instead of when a proof is submitted:
	// 1. Logic - Probabilistic proof allows claims to be settled (i.e. rewarded)
	//    even without a proof to be able to scale to unbounded Claims & Proofs.
	// 2. Implementation - This cannot be done from the `x/proof` module because
	//    it would create a circular dependency.
	settledResult, expiredResult, err := k.SettlePendingClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not settle pending claims due to error %v", err))
		return err
	}

	logger.Info(fmt.Sprintf(
		"settled %d claims and expired %d claims",
		settledResult.NumClaims,
		expiredResult.NumClaims,
	))

	// Update the relay mining difficulty for every service that settled pending
	// claims based on how many estimated relays were serviced for it.
	difficultyPerServiceMap, err := k.UpdateRelayMiningDifficulty(ctx, settledResult.RelaysPerServiceMap)
	if err != nil {
		logger.Error(fmt.Sprintf("could not update relay mining difficulty due to error %v", err))
		return err
	}
	logger.Info(fmt.Sprintf(
		"successfully updated the relay mining difficulty for %d services",
		len(settledResult.RelaysPerServiceMap),
	))

	// Telemetry - emit telemetry for each service's relay mining difficulty.
	for serviceId, newRelayMiningDifficulty := range difficultyPerServiceMap {
		var newRelayMiningTargetHash [protocol.RelayHasherSize]byte
		copy(newRelayMiningTargetHash[:], newRelayMiningDifficulty.TargetHash)

		// NB: The difficulty integer is just a human readable interpretation of
		// the target hash and is not actually used for business logic.
		difficulty := protocol.GetRelayDifficultyMultiplierToFloat32(newRelayMiningDifficulty.TargetHash)
		telemetry.RelayMiningDifficultyGauge(difficulty, serviceId)
		telemetry.RelayEMAGauge(newRelayMiningDifficulty.NumRelaysEma, serviceId)
	}

	return nil
}
