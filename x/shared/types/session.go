package types

// sessionGridAnchor resolves the session-grid anchor and the session number at that
// anchor for the given queryHeight, falling back to the genesis block-1 grid (anchor=1,
// numberAtAnchor=1) whenever the params do not describe the epoch that owns queryHeight.
//
// The fallback covers two cases (see #543 anchored-grid spec §3.4):
//   - anchor <= 0: the anchor is unset (pre-upgrade data / original genesis) → legacy grid.
//   - anchor > queryHeight: the params describe a LATER epoch than queryHeight. Go integer
//     division truncates toward zero, so a negative numerator (queryHeight - anchor) would
//     yield a garbage start height in the future. Fall back to the genesis grid instead.
//
// In the correct path, GetParamsAtHeight returns the params whose effective_height (= anchor)
// is the greatest value <= queryHeight, so anchor <= queryHeight holds and the math is exact.
func sessionGridAnchor(sharedParams *Params, queryHeight int64) (anchor, numberAtAnchor int64) {
	anchor = int64(sharedParams.GetSessionGridAnchorHeight())
	numberAtAnchor = int64(sharedParams.GetSessionNumberAtAnchor())
	if anchor <= 0 || anchor > queryHeight {
		// Unset, or params describe a later epoch than queryHeight → genesis block-1 grid.
		return 1, 1
	}
	if numberAtAnchor <= 0 {
		numberAtAnchor = 1
	}
	return anchor, numberAtAnchor
}

// GetSessionStartHeight returns the block height at which the session containing
// queryHeight starts, given the passed shared onchain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
//
// Boundaries are measured relative to the params epoch's session-grid anchor (#543), so
// that changing num_blocks_per_session does not misalign in-flight sessions. With anchor=1
// this reduces exactly to the legacy block-1 grid.
func GetSessionStartHeight(sharedParams *Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	anchor, _ := sessionGridAnchor(sharedParams, queryHeight)

	return anchor + ((queryHeight-anchor)/numBlocksPerSession)*numBlocksPerSession
}

// GetSessionEndHeight returns the block height at which the session containing
// queryHeight ends, given the passed shared onchain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndHeight(sharedParams *Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	sessionStartHeight := GetSessionStartHeight(sharedParams, queryHeight)

	return sessionStartHeight + numBlocksPerSession - 1
}

// GetSessionNumber returns the session number of the session containing queryHeight,
// given the passed onchain shared parameters.
// shared onchain parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
//
// Session numbers stay monotonic across epoch boundaries via session_number_at_anchor (#543);
// with anchor=1, numberAtAnchor=1 this reduces exactly to the legacy ((h-1)/N)+1 formula.
func GetSessionNumber(sharedParams *Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	anchor, numberAtAnchor := sessionGridAnchor(sharedParams, queryHeight)

	return numberAtAnchor + (queryHeight-anchor)/numBlocksPerSession
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session that includes queryHeight elapses, given the passed sharedParams.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
func GetSessionGracePeriodEndHeight(sharedParams *Params, queryHeight int64) int64 {
	sessionEndHeight := GetSessionEndHeight(sharedParams, queryHeight)
	return sessionEndHeight + int64(sharedParams.GetGracePeriodEndOffsetBlocks())
}

// IsGracePeriodElapsed returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func IsGracePeriodElapsed(sharedParams *Params, queryHeight, currentHeight int64) bool {
	return currentHeight > GetSessionGracePeriodEndHeight(sharedParams, queryHeight)
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens, for the provided sharedParams.
func GetClaimWindowOpenHeight(sharedParams *Params, queryHeight int64) int64 {
	sessionEndHeight := GetSessionEndHeight(sharedParams, queryHeight)
	claimWindowOpenOffsetBlocks := int64(sharedParams.GetClaimWindowOpenOffsetBlocks())
	// NB: An additional block (+1) is added to permit to relays arriving at the
	// last block of the session to be included in the claim before the smt is closed.
	return sessionEndHeight + claimWindowOpenOffsetBlocks + 1
}

// GetClaimWindowCloseHeight returns the block height at which the claim window of
// the session that includes queryHeight closes, for the provided sharedParams.
func GetClaimWindowCloseHeight(sharedParams *Params, queryHeight int64) int64 {
	claimWindowOpenHeight := GetClaimWindowOpenHeight(sharedParams, queryHeight)
	claimWindowCloseOffsetBlocks := int64(sharedParams.GetClaimWindowCloseOffsetBlocks())
	return claimWindowOpenHeight + claimWindowCloseOffsetBlocks
}

// GetProofWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens, given the passed sharedParams.
func GetProofWindowOpenHeight(sharedParams *Params, queryHeight int64) int64 {
	return GetClaimWindowCloseHeight(sharedParams, queryHeight) +
		int64(sharedParams.GetProofWindowOpenOffsetBlocks())
}

// GetProofWindowCloseHeight returns the block height at which the proof window of
// the session that includes queryHeight closes, given the passed sharedParams.
func GetProofWindowCloseHeight(sharedParams *Params, queryHeight int64) int64 {
	return GetProofWindowOpenHeight(sharedParams, queryHeight) +
		int64(sharedParams.GetProofWindowCloseOffsetBlocks())
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier
// and the passed sharedParams.
// TODO_TECHDEBT(@red-0ne): Having claim distribution windows was
// a requirement that was never determined to be necessary, but implemented regardless.
// We are keeping around the functions but TBD whether it is deemed necessary. The results
// of #711 are tangentially related to this requirement, after which the functions,
// helpers, comments and docs for claim distribution can either be repurposed or deleted.
func GetEarliestSupplierClaimCommitHeight(
	sharedParams *Params,
	queryHeight int64,
	claimWindowOpenBlockHash []byte,
	supplierOperatorAddr string,
) int64 {
	claimWindowOpenHeight := GetClaimWindowOpenHeight(sharedParams, queryHeight)

	// Generate a deterministic random (non-negative) int64, seeded by the claim
	// window open block hash and the supplier operator address.
	//randomNumber := poktrand.SeededInt63(claimWindowOpenBlockHash, []byte(supplierOperatorAddr))

	//distributionWindowSizeBlocks := sharedParams.GetClaimWindowCloseOffsetBlocks()
	//randCreateClaimHeightOffset := randomNumber % int64(distributionWindowSizeBlocks)

	//return claimWindowOpenHeight + randCreateClaimHeightOffset
	return claimWindowOpenHeight
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier
// and the passed sharedParams.
// TODO_TECHDEBT(@red-0ne): Having proof distribution windows was
// a requirement that was never determined to be necessary, but implemented regardless.
// We are keeping around the functions but TBD whether it is deemed necessary. The results
// of #711 are tangentially related to this requirement, after which the functions,
// helpers, comments and docs for claim distribution can either be repurposed or deleted.
func GetEarliestSupplierProofCommitHeight(
	sharedParams *Params,
	queryHeight int64,
	proofWindowOpenBlockHash []byte,
	supplierOperatorAddr string,
) int64 {
	proofWindowOpenHeight := GetProofWindowOpenHeight(sharedParams, queryHeight)

	// Generate a deterministic random (non-negative) int64, seeded by the proof
	// window open block hash and the supplier operator address.
	//randomNumber := poktrand.SeededInt63(proofWindowOpenBlockHash, []byte(supplierOperatorAddr))

	//distributionWindowSizeBlocks := sharedParams.GetProofWindowCloseOffsetBlocks()
	//randCreateProofHeightOffset := randomNumber % int64(distributionWindowSizeBlocks)

	//return proofWindowOpenHeight + randCreateProofHeightOffset
	return proofWindowOpenHeight
}

// GetNextSessionStartHeight returns the start block height of the session
// following the session that includes queryHeight, given the passed sharedParams.
func GetNextSessionStartHeight(sharedParams *Params, queryHeight int64) int64 {
	return GetSessionEndHeight(sharedParams, queryHeight) + 1
}

// IsSessionEndHeight returns true if the queryHeight is the last block of the session.
func IsSessionEndHeight(sharedParams *Params, queryHeight int64) bool {
	return queryHeight == GetSessionEndHeight(sharedParams, queryHeight)
}

// IsSessionStartHeight returns true if the height is the first block of the session.
func IsSessionStartHeight(sharedParams *Params, queryHeight int64) bool {
	return queryHeight == GetSessionStartHeight(sharedParams, queryHeight)
}

// GetSessionEndToProofWindowCloseBlocks returns the total number of blocks
// from the moment a session ends until the proof window closes.
func GetSessionEndToProofWindowCloseBlocks(params *Params) int64 {
	return int64(params.GetClaimWindowOpenOffsetBlocks() +
		params.GetClaimWindowCloseOffsetBlocks() +
		params.GetProofWindowOpenOffsetBlocks() +
		params.GetProofWindowCloseOffsetBlocks())
}

// GetSettlementSessionEndHeight returns the end height of the session in which the
// session that includes queryHeight is settled, given the passed shared onchain parameters.
func GetSettlementSessionEndHeight(sharedParams *Params, queryHeight int64) int64 {
	return GetSessionEndToProofWindowCloseBlocks(sharedParams) +
		GetSessionEndHeight(sharedParams, queryHeight) + 1
}

// GetNumPendingSessions returns the number of pending sessions (i.e. that have not
// yet been settled).
func GetNumPendingSessions(sharedParams *Params) int64 {
	// Get the number of blocks between the end of a session and the block height
	// at which the session claim is settled.
	numPendingSessionsBlocks := GetSessionEndToProofWindowCloseBlocks(sharedParams)
	// Use the number of blocks per session to calculate the number of pending sessions.
	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	// numBlocksPerSession - 1 is added to round up the integer division so that pending
	// sessions are all the sessions that have their end height at least `pendingBlocks` old.
	return (numPendingSessionsBlocks + numBlocksPerSession - 1) / numBlocksPerSession
}
