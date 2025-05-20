package types

// GetSessionStartHeight returns the block height at which the session containing
// queryHeight starts, given the passed shared onchain parameters updates history.
// Returns 0 if the block height is not a consensus produced block.
// By using the params updates history it ensures accurate session boundary
// calculations even when session lengths change.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetSessionStartHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	// Get the effective params update as of the query height.
	// This is the params values that were in effect at the time of the query.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	// Since the params always update at the start of a session, we can use the
	// effective block height as a starting point to calculate the session start height.
	paramsUpdateFirstSessionStartHeight := sharedParamsUpdate.ActivationHeight
	// Calculate the height of the query height relative to the params update effective
	// height which is a session start height.
	relativeHeight := queryHeight - paramsUpdateFirstSessionStartHeight
	numBlocksPerSession := int64(sharedParamsUpdate.Params.GetNumBlocksPerSession())

	return paramsUpdateFirstSessionStartHeight + (relativeHeight/numBlocksPerSession)*numBlocksPerSession
}

// GetSessionEndHeight returns the block height at which the session containing
// queryHeight ends, given the passed shared onchain parameters updates history.
// Returns 0 if the block height is not a consensus produced block.
// By using the params updates history it ensures accurate session boundary
// calculations even when session lengths change.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	// Get the numBlocksPerSession of the effective params update as of the query height
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)
	numBlocksPerSession := int64(sharedParamsUpdate.Params.GetNumBlocksPerSession())

	// Get the session start height first
	sessionStartHeight := GetSessionStartHeight(sharedParamsUpdates, queryHeight)

	return sessionStartHeight + numBlocksPerSession - 1
}

// GetSessionNumber returns the session number of the session containing queryHeight,
// given the passed onchain shared parameters updates history.
// By using the params updates history it ensures accurate session boundary
// calculations even when session lengths change.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetSessionNumber(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	// Start with session 1, block 1
	sessionNum := int64(1)
	currentHeight := int64(1)

	// Process all parameter updates prior to query height
	for i, update := range sharedParamsUpdates {
		updateHeight := update.ActivationHeight

		// Skip updates after our query height
		if updateHeight > queryHeight {
			break
		}

		// Calculate sessions completed with previous parameters
		if i > 0 {
			prevBlocksPerSession := int64(sharedParamsUpdates[i-1].Params.GetNumBlocksPerSession())
			completeSessions := (updateHeight - currentHeight) / prevBlocksPerSession
			sessionNum += completeSessions
		}

		// Update current height to this parameter update
		currentHeight = updateHeight
	}

	// Calculate sessions from the last parameter update to query height
	lastUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)
	lastBlocksPerSession := int64(lastUpdate.Params.GetNumBlocksPerSession())

	completeSessions := (queryHeight - currentHeight) / lastBlocksPerSession
	sessionNum += completeSessions

	return sessionNum
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session that includes queryHeight elapses, given the passed sharedParams
// updates history.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
func GetSessionGracePeriodEndHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	// Use the params' GracePeriodEndOffsetBlocks effective as of queryHeight to
	// calculate the grace period end height.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	sessionEndHeight := GetSessionEndHeight(sharedParamsUpdates, queryHeight)
	return sessionEndHeight + int64(sharedParamsUpdate.Params.GetGracePeriodEndOffsetBlocks())
}

// IsGracePeriodElapsed returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func IsGracePeriodElapsed(sharedParamsUpdates []*ParamsUpdate, queryHeight, currentHeight int64) bool {
	return currentHeight > GetSessionGracePeriodEndHeight(sharedParamsUpdates, queryHeight)
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of the
// session that includes queryHeight opens, for the provided sharedParams updates history.
func GetClaimWindowOpenHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	// Use the params' GetClaimWindowOpenOffsetBlocks effective as of queryHeight to
	// calculate the claim window open height.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	sessionEndHeight := GetSessionEndHeight(sharedParamsUpdates, queryHeight)
	claimWindowOpenOffsetBlocks := int64(sharedParamsUpdate.Params.GetClaimWindowOpenOffsetBlocks())
	// NB: An additional block (+1) is added to permit to relays arriving at the
	// last block of the session to be included in the claim before the smt is closed.
	return sessionEndHeight + claimWindowOpenOffsetBlocks + 1
}

// GetClaimWindowCloseHeight returns the block height at which the claim window of the
// session that includes queryHeight closes, for the provided sharedParams updates history.
func GetClaimWindowCloseHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	// Use the params' GetClaimWindowCloseOffsetBlocks effective as of queryHeight to
	// calculate the claim window close height.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	claimWindowOpenHeight := GetClaimWindowOpenHeight(sharedParamsUpdates, queryHeight)
	claimWindowCloseOffsetBlocks := int64(sharedParamsUpdate.Params.GetClaimWindowCloseOffsetBlocks())
	return claimWindowOpenHeight + claimWindowCloseOffsetBlocks
}

// GetProofWindowOpenHeight returns the block height at which the claim window of the
// session that includes queryHeight opens, given the passed sharedParams updates history.
func GetProofWindowOpenHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	// Use the params' GetProofWindowOpenOffsetBlocks effective as of queryHeight to
	// calculate the proof window open height.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	return GetClaimWindowCloseHeight(sharedParamsUpdates, queryHeight) +
		int64(sharedParamsUpdate.Params.GetProofWindowOpenOffsetBlocks())
}

// GetProofWindowCloseHeight returns the block height at which the proof window of the
// session that includes queryHeight closes, given the passed sharedParams updates history.
func GetProofWindowCloseHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	// Use the params' GetProofWindowCloseOffsetBlocks effective as of queryHeight to
	// calculate the proof window close height.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	return GetProofWindowOpenHeight(sharedParamsUpdates, queryHeight) +
		int64(sharedParamsUpdate.Params.GetProofWindowCloseOffsetBlocks())
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier
// and the passed sharedParams updates history.
// TODO_TECHDEBT(@red-0ne): Having claim distribution windows was
// a requirement that was never determined to be necessary, but implemented regardless.
// We are keeping around the functions but TBD whether it is deemed necessary. The results
// of #711 are tengentially related to this requirement, after which the functions,
// helpers, comments and docs for claim distribution can either be repurposed or deleted.
func GetEarliestSupplierClaimCommitHeight(
	sharedParamsUpdates ParamsHistory,
	queryHeight int64,
	claimWindowOpenBlockHash []byte,
	supplierOperatorAddr string,
) int64 {
	claimWindowOpenHeight := sharedParamsUpdates.GetClaimWindowOpenHeight(queryHeight)

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
// and the passed sharedParams updates history.
// TODO_TECHDEBT(@red-0ne): Having proof distribution windows was
// a requirement that was never determined to be necessary, but implemented regardless.
// We are keeping around the functions but TBD whether it is deemed necessary. The results
// of #711 are tengentially related to this requirement, after which the functions,
// helpers, comments and docs for claim distribution can either be repurposed or deleted.
func GetEarliestSupplierProofCommitHeight(
	sharedParamsUpdates ParamsHistory,
	queryHeight int64,
	proofWindowOpenBlockHash []byte,
	supplierOperatorAddr string,
) int64 {
	proofWindowOpenHeight := sharedParamsUpdates.GetProofWindowOpenHeight(queryHeight)

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
func GetNextSessionStartHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	return GetSessionEndHeight(sharedParamsUpdates, queryHeight) + 1
}

// IsSessionEndHeight returns true if the queryHeight is the last block of the session.
func IsSessionEndHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) bool {
	return queryHeight == GetSessionEndHeight(sharedParamsUpdates, queryHeight)
}

// IsSessionStartHeight returns true if the height is the first block of the session.
func IsSessionStartHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) bool {
	return queryHeight == GetSessionStartHeight(sharedParamsUpdates, queryHeight)
}

// GetSessionEndToProofWindowCloseBlocks returns the total number of blocks
// from the moment a session ends until the proof window closes.
func GetSessionEndToProofWindowCloseBlocks(params *Params) int64 {
	return int64(params.GetClaimWindowOpenOffsetBlocks() +
		params.GetClaimWindowCloseOffsetBlocks() +
		params.GetProofWindowOpenOffsetBlocks() +
		params.GetProofWindowCloseOffsetBlocks())
}

// GetSettlementSessionEndHeight returns the end height of the session in which the session
// that includes queryHeight is settled, given the passed shared onchain parameters updates history.
func GetSettlementSessionEndHeight(sharedParamsUpdates []*ParamsUpdate, queryHeight int64) int64 {
	// Use the params' GetProofWindowCloseOffsetBlocks effective as of queryHeight to
	// calculate the proof window close height.
	sharedParamsUpdate := GetActiveParamsUpdate(sharedParamsUpdates, queryHeight)

	return GetSessionEndToProofWindowCloseBlocks(&sharedParamsUpdate.Params) +
		GetSessionEndHeight(sharedParamsUpdates, queryHeight) + 1
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

type paramUpdate[T any] interface {
	GetParams() T
	GetActivationHeight() int64
	GetDeactivationHeight() int64
}

// GetActiveParamsUpdate returns the effective params update as of the query height.
func GetActiveParamsUpdate[V any, T paramUpdate[V]](sharedParamsUpdates []T, queryHeight int64) T {
	var effectiveParamsUpdate T
	for _, update := range sharedParamsUpdates {
		// The params updates are chronologically ordered from the oldest to the most recent.
		// We can stop iterating when we find the first params update that is effective
		// after the query height.
		if update.GetActivationHeight() > queryHeight {
			break
		}

		effectiveParamsUpdate = update
	}

	return effectiveParamsUpdate
}

func GetCurrentParams[V any, T paramUpdate[V]](sharedParamsUpdates []T, queryHeight int64) V {
	return GetActiveParamsUpdate(sharedParamsUpdates, queryHeight).GetParams()
}
