package types

import (
	"context"
	"math/big"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// IsClaimed returns true if the MorseClaimableAccount has been claimed;
// i.e. ShannonDestAddress is not empty OR the ClaimedAtHeight is greater than 0.
func (m *MorseClaimableAccount) IsClaimed() bool {
	return m.ShannonDestAddress != "" || m.ClaimedAtHeight > 0
}

// IsUnbonding indicates that the MorseClaimableAccount began unbonding on Morse
// but its unbonding period has NOT yet elapsed at the time that the Morse snapshot was taken.
func (m *MorseClaimableAccount) IsUnbonding() bool {
	// DEV_NOTE: The UnstakingTime field is a time.Time type, which has a zero value of "0001-01-01T00:00:00Z" when printed as an ISO8601 string.
	// See: https://pkg.go.dev/time#Time.IsZero
	return !m.UnstakingTime.IsZero()
}

// HasUnbonded indicates that:
// 1. the MorseClaimableAccount began unbonding on Morse
// 2. The unbonding period has elapsed.
// For example, the supplier was claimed on Shannon > 21 days after it began unbonding on Morse.
func (m *MorseClaimableAccount) HasUnbonded(ctx context.Context) bool {
	return m.IsUnbonding() && m.SecondsUntilUnbonded(ctx) <= 0
}

// SecondsUntilUnbonded returns the number of seconds until the MorseClaimableAccount's
// unbonding period will elapse.
func (m *MorseClaimableAccount) SecondsUntilUnbonded(ctx context.Context) int64 {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	absoluteUnstakingTime := m.UnstakingTime
	absoluteBlockTime := sdkCtx.BlockTime()
	durationUntilUnbonded := absoluteUnstakingTime.Sub(absoluteBlockTime)
	secondsUntilUnbonded := durationUntilUnbonded.Seconds()
	return int64(secondsUntilUnbonded)
}

// GetEstimatedUnbondingEndHeight returns the estimated block height at which the
// MorseClaimableAccount's unbonding period will end. The estimation process includes:
//
// The estimation process includes:
// - Calculating the remaining time until unstaking is complete.
// - Using the estimated block duration from an off-chain configuration.
//
// Returns:
// - The estimated block height when unbonding will end.
// - true if the unbonding period has not yet elapsed.
func (m *MorseClaimableAccount) GetEstimatedUnbondingEndHeight(
	ctx context.Context,
	sharedParams sharedtypes.Params,
) (height int64, isUnbonded bool) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Retrieve the estimated block duration for the current chain from a lookup table.
	// DEV_NOTE: This is an offchain config value; i.e. not queryable.
	estimatedBlockDuration, ok := pocket.EstimatedBlockDurationByChainId[sdkCtx.ChainID()]
	if !ok || estimatedBlockDuration == 0 {
		// Defensive: avoid division by zero and clarify error state
		return -1, false
	}

	// Check if unstaking is complete:
	//   - Calculate the remaining duration until unstaking.
	//   - If the duration is zero or negative, the unstaking period has elapsed.
	//   - Return -1 to indicate that unbonding is complete.
	secondsUntilUnstakeCompletion := m.UnstakingTime.Sub(sdkCtx.BlockTime()).Seconds()
	if secondsUntilUnstakeCompletion <= 0 {
		return -1, true
	}

	// Calculate the estimated Shannon unstake session end height.
	// I.e. the end height of the session after which the claimed
	// Shannon supplier will be unstaked.
	estimatedBlocksUntilUnstakeCompletion := big.NewRat(int64(secondsUntilUnstakeCompletion), int64(estimatedBlockDuration))
	estimatedUnstakeCompletionHeightRat := new(big.Rat).Add(
		big.NewRat(sdkCtx.BlockHeight(), 1),
		estimatedBlocksUntilUnstakeCompletion,
	)
	estimatedUnstakeCompletionHeight := new(big.Int).Div(
		estimatedUnstakeCompletionHeightRat.Num(),
		estimatedUnstakeCompletionHeightRat.Denom(),
	).Int64()
	expectedUnstakeSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, estimatedUnstakeCompletionHeight)

	return expectedUnstakeSessionEndHeight, false
}
