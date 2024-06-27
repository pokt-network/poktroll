package tokenomics

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

// EndBlocker called at every block and settles all pending claims.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) (err error) {
	logger := k.Logger().With("method", "EndBlocker")
	// NB: There are two main reasons why we settle expiring claims in the end
	// instead of when a proof is submitted:
	// 1. Logic - Probabilistic proof allows claims to be settled (i.e. rewarded)
	//    even without a proof to be able to scale to unbounded Claims & Proofs.
	// 2. Implementation - This cannot be done from the `x/proof` module because
	//    it would create a circular dependency.
	numClaimsSettled,
		numClaimsExpired,
		relaysPerServiceMap,
		computeUnitsPerServiceMap,
		err := k.SettlePendingClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not settle pending claims due to error %v", err))
		return err
	}

	// Accumulate compute units for metrics.
	// TODO_IMPROVE(@bryanchriswhite, @red-0ne): It would be preferable to have telemetry
	// counter functions return an "event" or "event set", similar to how polylog/zerolog work.
	var numComputeUnits uint64
	for _, serviceComputeUnits := range computeUnitsPerServiceMap {
		numComputeUnits += serviceComputeUnits
	}

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		telemetry.ClaimComputeUnitsCounter(
			prooftypes.ClaimProofStage_SETTLED,
			numComputeUnits,
			err,
		)
		telemetry.ClaimCounter(
			prooftypes.ClaimProofStage_SETTLED,
			numClaimsSettled,
			err,
		)
		telemetry.ClaimCounter(
			prooftypes.ClaimProofStage_EXPIRED,
			numClaimsExpired,
			err,
		)
		// TODO_IMPROVE(#observability): Add a counter for expired compute units.
	}()

	logger.Info(fmt.Sprintf("settled %d claims and expired %d claims", numClaimsSettled, numClaimsExpired))

	// Update the relay mining difficulty for every service that settled pending
	// claims based on how many estimated relays were serviced for it.
	err = k.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	if err != nil {
		logger.Error(fmt.Sprintf("could not update relay mining difficulty due to error %v", err))
		return err
	}
	logger.Info(fmt.Sprintf("successfully updated the relay mining difficulty for %d services", len(relaysPerServiceMap)))

	return nil
}
