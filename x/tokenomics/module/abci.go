package tokenomics

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

// EndBlocker called at every block and settles all pending claims.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	logger := k.Logger().With("method", "EndBlocker")
	// NB: There are two main reasons why we settle expiring claims in the end
	// instead of when a proof is submitted:
	// 1. Logic - Probabilistic proof allows claims to be settled (i.e. rewarded)
	//    even without a proof to be able to scale to unbounded Claims & Proofs.
	// 2. Implementation - This cannot be done from the `x/proof` module because
	//    it would create a circular dependency.
	numClaimsSettled, numClaimsExpired, relaysPerServiceMap, err := k.SettlePendingClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not settle pending claims due to error %v", err))
		return err
	}

	defer telemetry.ClaimCounter(
		telemetry.ClaimProofStageSettled,
		func() uint64 { return numClaimsSettled },
	)

	defer telemetry.ClaimCounter(
		telemetry.ClaimProofStageExpired,
		func() uint64 { return numClaimsExpired },
	)

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
