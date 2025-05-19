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

// IsUnbonding indicates that the MorseClaimableAccount began unbonding on Morse
// but its unbonding peroid has NOT yet elapsed.
func (m *MorseClaimableAccount) IsUnbonding() bool {
	// DEV_NOTE: The UnstakingTime field is a time.Time type, which has a zero value of "0001-01-01T00:00:00Z" when printed as an ISO8601 string.
	// See: https://pkg.go.dev/time#Time.IsZero
	return !m.UnstakingTime.IsZero()
}

// HasUnbonded indicates that the MorseClaimableAccount began unbonding on Morse
// and the unbonding period has elapsed.
func (m *MorseClaimableAccount) HasUnbonded() bool {
	return m.IsUnbonding() && m.SecondsUntilUnbonded() <= 0
}

// SecondsUntilUnbonded returns the number of seconds until the MorseClaimableAccount's
// unbonding period will elapse.
func (m *MorseClaimableAccount) SecondsUntilUnbonded() int64 {
	return int64(time.Until(m.UnstakingTime).Seconds())
}

// TODO_IN_THIS_COMMIT: godoc...
// ... returns -1 if unstaking is already complete...
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
