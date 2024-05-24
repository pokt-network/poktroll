package tokenomics

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

// EndBlocker called at every block and settles all pending claims.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	blockHeight := ctx.BlockHeight()
	logger := k.Logger().With(
		"method", "EndBlocker",
		"blockHeight", blockHeight,
	)

	// There are two main reasons why we settle expiring claims in the end
	// instead of when a proof is submitted:
	// 1. Logic - Probabilistic proof allows claims to be settled (i.e. rewarded)
	//    even without a proof to be able to scale to unbounded Claims & Proofs.
	// 2. Implementation - This cannot be done from the `x/proof` module because
	//    it would create a circular dependency.
	numClaimsSettled, numClaimsExpired, relaysPerServiceMap, err := k.SettlePendingClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to settle pending claims: %v", err))
		return err
	}
	logger.Info(fmt.Sprintf("settled %d and expired %d claims"), numClaimsSettled, numClaimsExpired)

	// Update relay mining difficulty based on the amount of on-chain volume
	// TODO_IN_THIS_PR: Discuss if we should do this on every endBlocker or periodically.
	err = k.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to update relay mining difficulty: %v", err))
	}

	return err
}
