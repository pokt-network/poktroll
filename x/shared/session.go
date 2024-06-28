package shared

import (
	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	minimumClaimWindowSizeBlocks = 1
	minimumProofWindowSizeBlocks = 1
)

// TODO_DOCUMENT(@bryanchriswhite): Move this into the documentation: https://github.com/pokt-network/poktroll/pull/571#discussion_r1630923625

// SessionGracePeriodBlocks is the number of blocks after the session ends before the
// "session grace period" is considered to have elapsed.
//
// TODO_BLOCKER(@bryanchriswhite): This is a place-holder that will be removed
// once the respective governance parameter is implemented.
const SessionGracePeriodBlocks = 2

// GetSessionStartHeight returns the block height at which the session containing
// queryHeight starts, given the passed shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetSessionStartHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())

	// TODO_BLOCKER(@bryanchriswhite, #543): If the num_blocks_per_session param has ever been changed,
	// this function may cause unexpected behavior.
	return queryHeight - ((queryHeight - 1) % numBlocksPerSession)
}

// GetSessionEndHeight returns the block height at which the session containing
// queryHeight ends, given the passed shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	sessionStartHeight := GetSessionStartHeight(sharedParams, queryHeight)

	return sessionStartHeight + numBlocksPerSession - 1
}

// GetSessionNumber returns the session number of the session containing queryHeight,
// given the passed on-chain shared parameters.
// shared on-chain parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetSessionNumber(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())

	// TODO_BLOCKER(@bryanchriswhite, #543): If the num_blocks_per_session param has ever been changed,
	// this function may cause unexpected behavior.
	return ((queryHeight - 1) / numBlocksPerSession) + 1
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session that includes queryHeight elapses, given the passed sharedParams.
func GetSessionGracePeriodEndHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	sessionEndHeight := GetSessionEndHeight(sharedParams, queryHeight)
	return sessionEndHeight + SessionGracePeriodBlocks
}

// IsGracePeriodElapsed returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func IsGracePeriodElapsed(sharedParams *sharedtypes.Params, queryHeight, currentHeight int64) bool {
	return currentHeight > GetSessionGracePeriodEndHeight(sharedParams, queryHeight)
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens, for the provided sharedParams.
func GetClaimWindowOpenHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	sessionEndHeight := GetSessionEndHeight(sharedParams, queryHeight)
	claimWindowOpenOffsetBlocks := int64(sharedParams.GetClaimWindowOpenOffsetBlocks())
	// NB: An additional block (+1) is added to permit to relays arriving at the
	// last block of the session to be included in the claim before the smt is closed.
	return sessionEndHeight + claimWindowOpenOffsetBlocks + 1
}

// GetClaimWindowCloseHeight returns the block height at which the claim window of
// the session that includes queryHeight closes, for the provided sharedParams.
func GetClaimWindowCloseHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	claimWindowOpenHeight := GetClaimWindowOpenHeight(sharedParams, queryHeight)
	claimWindowCloseOffsetBlocks := int64(sharedParams.GetClaimWindowCloseOffsetBlocks())
	return claimWindowOpenHeight + claimWindowCloseOffsetBlocks
}

// GetProofWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens, given the passed sharedParams.
func GetProofWindowOpenHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	return GetClaimWindowCloseHeight(sharedParams, queryHeight) +
		int64(sharedParams.GetProofWindowOpenOffsetBlocks())
}

// GetProofWindowCloseHeight returns the block height at which the proof window of
// the session that includes queryHeight closes, given the passed sharedParams.
func GetProofWindowCloseHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	return GetProofWindowOpenHeight(sharedParams, queryHeight) +
		int64(sharedParams.GetProofWindowCloseOffsetBlocks())
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier
// and the passed sharedParams.
func GetEarliestSupplierClaimCommitHeight(
	sharedParams *sharedtypes.Params,
	queryHeight int64,
	claimWindowOpenBlockHash []byte,
	supplierAddr string,
) int64 {
	claimWindowOpenHeight := GetClaimWindowOpenHeight(sharedParams, queryHeight)

	// Generate a deterministic random (non-negative) int64, seeded by the claim
	// window open block hash and the supplier address.
	randomNumber := poktrand.SeededInt63(claimWindowOpenBlockHash, []byte(supplierAddr))

	distributionWindowSizeBlocks := GetClaimWindowSizeBlocks(sharedParams)
	randCreateClaimHeightOffset := randomNumber % int64(distributionWindowSizeBlocks)

	return claimWindowOpenHeight + randCreateClaimHeightOffset
}

// GetClaimWindowSizeBlocks returns the number of blocks between the opening and closing
// of the claim window, given the passed sharedParams.
func GetClaimWindowSizeBlocks(sharedParams *sharedtypes.Params) uint64 {
	windowSizeBlocks := sharedParams.ClaimWindowCloseOffsetBlocks -
		sharedParams.ClaimWindowOpenOffsetBlocks -
		minimumClaimWindowSizeBlocks

	if windowSizeBlocks < 1 {
		return 1
	}

	return windowSizeBlocks
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier
// and the passed sharedParams.
func GetEarliestSupplierProofCommitHeight(
	sharedParams *sharedtypes.Params,
	queryHeight int64,
	proofWindowOpenBlockHash []byte,
	supplierAddr string,
) int64 {
	proofWindowOpenHeight := GetProofWindowOpenHeight(sharedParams, queryHeight)

	// Generate a deterministic random (non-negative) int64, seeded by the proof
	// window open block hash and the supplier address.
	randomNumber := poktrand.SeededInt63(proofWindowOpenBlockHash, []byte(supplierAddr))

	distributionWindowSizeBlocks := GetProofWindowSizeBlocks(sharedParams)
	randCreateProofHeightOffset := randomNumber % int64(distributionWindowSizeBlocks)

	return proofWindowOpenHeight + randCreateProofHeightOffset
}

// GetProofWindowSizeBlocks returns the number of blocks between the opening and closing
// of the proof window, given the passed sharedParams.
func GetProofWindowSizeBlocks(sharedParams *sharedtypes.Params) uint64 {
	windowSizeBlocks := sharedParams.ProofWindowCloseOffsetBlocks -
		sharedParams.ProofWindowOpenOffsetBlocks -
		minimumProofWindowSizeBlocks

	if windowSizeBlocks < 1 {
		return 1
	}

	return windowSizeBlocks
}
