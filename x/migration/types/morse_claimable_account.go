package types

import (
	"context"
	"math/big"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
)

// IsClaimed returns true if the MorseClaimableAccount has been claimed;
// i.e. ShannonDestAddress is not empty OR the ClaimedAtHeight is greater than 0.
func (m *MorseClaimableAccount) IsClaimed() bool {
	return m.ShannonDestAddress != "" || m.ClaimedAtHeight > 0
}

// TODO_IN_THIS_COMMIT: godoc...
func (m *MorseClaimableAccount) GetEstimatedUnbondingEndHeight(ctx context.Context) int64 {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Retrieve the estimated block duration for the current chain from a lookup table.
	// DEV_NOTE: This is an offchain config value; i.e. not queryable.
	estimatedBlockDuration := int64(pocket.EstimatedBlockDurationByChainId[sdkCtx.ChainID()])

	// TODO_IN_THIS_COMMIT: comment...
	// ... return early if unstaking is already complete...
	durationUntilUnstakeCompletion := int64(time.Until(m.UnstakingTime))
	if durationUntilUnstakeCompletion <= 0 {
		// The unstaking completion time has already elapsed.
		return -1
	}

	// Calculated the estimated Shannon unstake session end height.
	// I.e. the end height of the session after which the claimed
	// Shannon supplier will be unstaked.
	estimatedBlocksUntilUnstakeCompletion := big.NewRat(durationUntilUnstakeCompletion, estimatedBlockDuration)
	estimatedUnstakeCompletionHeight := new(big.Rat).Add(
		big.NewRat(sdkCtx.BlockHeight(), 1),
		estimatedBlocksUntilUnstakeCompletion,
	)
	return estimatedUnstakeCompletionHeight.Num().Int64()
}
