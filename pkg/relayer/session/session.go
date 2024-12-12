package session

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ relayer.RelayerSessionsManager = (*relayerSessionsManager)(nil)

// sessionTreesMap is an alias type for a map of
// sessionEndHeight->sessionId->supplierOperatorAddress->SessionTree.
// It is used to keep track of the sessions that are created in the RelayMiner
// by grouping them by their end block height, session id and supplier operator address.
type sessionsTreesMap = map[int64]map[string]map[string]relayer.SessionTree

// relayerSessionsManager is an implementation of the RelayerSessions interface.
// TODO_TEST: Add tests to the relayerSessionsManager.
type relayerSessionsManager struct {
	logger polylog.Logger

	relayObs relayer.MinedRelaysObservable

	// sessionTrees is a map of blockHeight->sessionId->supplierOperatorAddress->sessionTree.
	// The block height index is used to know when the sessions contained in the
	// entry should be closed, this helps to avoid iterating over all sessionsTrees
	// to check if they are ready to be closed.
	// The sessionTrees are grouped by supplierOperatorAddress since each supplier has to
	// claim the work it has been assigned.
	sessionsTrees   sessionsTreesMap
	sessionsTreesMu *sync.Mutex

	// blockClient is used to get the notifications of committed blocks.
	blockClient client.BlockClient

	// supplierClients is used to create claims and submit proofs for sessions.
	supplierClients *supplier.SupplierClientMap

	// storesDirectory points to a path on disk where KVStore data files are created.
	storesDirectory string

	// sharedQueryClient is used to query shared module parameters.
	sharedQueryClient client.SharedQueryClient

	// serviceQueryClient is used to query for a service with a given ID.
	// This is used to get the ComputeUnitsPerRelay, which is used as the weight of a mined relay
	// when adding a mined relay to a session's tree.
	serviceQueryClient client.ServiceQueryClient

	// proofQueryClient is used to query for the proof requirement threshold and
	// requirement probability governance parameters to determine whether a submitted
	// claim requires a proof.
	proofQueryClient client.ProofQueryClient

	// bankQueryClient is used to query for the bank module parameters.
	bankQueryClient client.BankQueryClient
}

// NewRelayerSessions creates a new relayerSessions.
//
// Required dependencies:
//   - client.BlockClient
//   - client.SupplierClientMap
//   - client.ProofQueryClient
//
// Available options:
//   - WithStoresDirectory
//   - WithSigningKeyNames
func NewRelayerSessions(
	ctx context.Context,
	deps depinject.Config,
	opts ...relayer.RelayerSessionsManagerOption,
) (_ relayer.RelayerSessionsManager, err error) {
	rs := &relayerSessionsManager{
		logger:          polylog.Ctx(ctx),
		sessionsTrees:   make(sessionsTreesMap),
		sessionsTreesMu: &sync.Mutex{},
	}

	if err := depinject.Inject(
		deps,
		&rs.blockClient,
		&rs.supplierClients,
		&rs.sharedQueryClient,
		&rs.serviceQueryClient,
		&rs.proofQueryClient,
		&rs.bankQueryClient,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(rs)
	}

	if err := rs.validateConfig(); err != nil {
		return nil, err
	}

	return rs, nil
}

// Start maps over the session trees at the end of each, respective, session.
// The session trees are piped through a series of map operations which progress
// them through the claim/proof lifecycle, broadcasting transactions to  the
// network as necessary.
// It IS NOT BLOCKING as map operations run in their own goroutines.
func (rs *relayerSessionsManager) Start(ctx context.Context) {
	// NB: must cast back to generic observable type to use with Map.
	// relayer.MinedRelaysObservable cannot be an alias due to gomock's lack of
	// support for generic types.
	relayObs := observable.Observable[*relayer.MinedRelay](rs.relayObs)

	// Map eitherMinedRelays to a new observable of an error type which is
	// notified if an error was encountered while attempting to add the relay to
	// the session tree.
	miningErrorsObs := channel.Map(ctx, relayObs, rs.mapAddMinedRelayToSessionTree)
	logging.LogErrors(ctx, miningErrorsObs)

	// Start claim/proof pipeline for each supplier that is present in the RelayMiner.
	for supplierOperatorAddress, supplierClient := range rs.supplierClients.SupplierClients {
		supplierSessionsToClaimObs := rs.supplierSessionsToClaim(ctx, supplierOperatorAddress)
		claimedSessionsObs := rs.createClaims(ctx, supplierClient, supplierSessionsToClaimObs)
		rs.submitProofs(ctx, supplierClient, claimedSessionsObs)
	}
}

// Stop unsubscribes all observables from the InsertRelays observable which
// will close downstream observables as they drain.
//
// TODO_TECHDEBT: Either add a mechanism to wait for draining to complete
// and/or ensure that the state at each pipeline stage is persisted to disk
// and exit as early as possible.
func (rs *relayerSessionsManager) Stop() {
	rs.relayObs.UnsubscribeAll()
}

// SessionsToClaim returns an observable that notifies when sessions are ready to be claimed.
func (rs *relayerSessionsManager) InsertRelays(relays relayer.MinedRelaysObservable) {
	rs.relayObs = relays
}

// ensureSessionTree returns the SessionTree for the session and supplier
// corresponding to the relay request metadata.
// If no tree for the session exists, a new SessionTree is created before returning.
func (rs *relayerSessionsManager) ensureSessionTree(
	relayRequestMetadata *servicetypes.RelayRequestMetadata,
) (relayer.SessionTree, error) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	// Get the sessions that end at the relay request's sessionEndHeight.
	sessionHeader := relayRequestMetadata.SessionHeader
	sessionTreesWithEndHeight, ok := rs.sessionsTrees[sessionHeader.SessionEndBlockHeight]

	// If there is no map for sessions at the sessionEndHeight, create one.
	if !ok {
		sessionTreesWithEndHeight = make(map[string]map[string]relayer.SessionTree)
		rs.sessionsTrees[sessionHeader.SessionEndBlockHeight] = sessionTreesWithEndHeight
	}

	// Get the sessionTreeWithSessionId for the given session.
	sessionTreeWithSessionId, ok := sessionTreesWithEndHeight[sessionHeader.SessionId]

	// If there is no map for session trees with the session id, create one.
	if !ok {
		sessionTreeWithSessionId = make(map[string]relayer.SessionTree)
		sessionTreesWithEndHeight[sessionHeader.SessionId] = sessionTreeWithSessionId
	}

	supplierOperatorAccAddress, err := cosmostypes.AccAddressFromBech32(relayRequestMetadata.SupplierOperatorAddress)
	if err != nil {
		return nil, err
	}
	supplierOperatorAddress := supplierOperatorAccAddress.String()

	// Get the sessionTree for the supplier corresponding to the relay request.
	sessionTree, ok := sessionTreeWithSessionId[supplierOperatorAddress]

	// If the sessionTree does not exist, create and assign it to the
	// sessionTreeWithSessionId map for the given supplier operator address.
	if !ok {
		sessionTree, err = NewSessionTree(sessionHeader, &supplierOperatorAccAddress, rs.storesDirectory, rs.logger)
		if err != nil {
			return nil, err
		}

		sessionTreeWithSessionId[supplierOperatorAddress] = sessionTree
	}

	return sessionTree, nil
}

// forEachBlockClaimSessionsFn returns a new ForEachFn that sends a lists of sessions which
// are eligible to be claimed at each block height on sessionsToClaimsPublishCh, effectively
// mapping committed blocks to a list of sessions which can be claimed as of that block.
//
// forEachBlockClaimSessionsFn returns a new ForEachFn that is called once for each block height.
// Given the current sessions in the rs.sessionsTrees map at the time of each call, it:
// - fetches the current shared module params
// - builds a list of "on-time" & "late" sessions that are eligible to be claimed as of a given block height
// - sends "late" & "on-time" sessions on sessionsToClaimsPublishCh as distinct notifications
//
// If "late" sessions are found, they are emitted as quickly as possible and are expected
// to bypass downstream delay operations. "late" sessions are emitted, as they're discovered
// (by iterating over map keys).
//
// Under nominal conditions, only one set of "on-time" sessions (w/ the same session start/end heights)
// should be present in the rs.sessionsTrees map. "Late" sessions
// are expected to present in the presence of network interruptions, restarts, or other
// disruptions to the relayminer process.
// TODO_IMPROVE: Add the ability for the process to resume where it left off in
// case the process is restarted or the connection is dropped and reconnected.
func (rs *relayerSessionsManager) forEachBlockClaimSessionsFn(
	sessionsSupplier string,
	sessionsToClaimsPublishCh chan<- []relayer.SessionTree,
) channel.ForEachFn[client.Block] {
	return func(ctx context.Context, block client.Block) {
		rs.sessionsTreesMu.Lock()
		defer rs.sessionsTreesMu.Unlock()

		// onTimeSessions are the sessions that are still within their grace period.
		// They are on time and will wait for their create claim window to open.
		// They will be emitted last, after all the late sessions have been emitted.
		var onTimeSessions []relayer.SessionTree

		// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
		// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
		// to get the most recently (asynchronously) observed (and cached) value.
		sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
		if err != nil {
			rs.logger.Error().Err(err).Msg("unable to query shared module params")
			return
		}

		// Check if there are sessions that need to enter the claim/proof phase as their
		// end block height was the one before the last committed block or earlier.
		// Iterate over the sessionsTrees map to get the ones that end at a block height
		// lower than the current block height.
		for sessionEndHeight, sessionsTreesEndingAtBlockHeight := range rs.sessionsTrees {
			// Late sessions are the ones that have their session grace period elapsed
			// and should already have been claimed.
			// Group them by their end block height and emit each group separately
			// before emitting the on-time sessions.
			var lateSessions []relayer.SessionTree

			claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

			// Checking for sessions to claim with <= operator,
			// which means that it would include sessions that were supposed to be
			// claimed in previous block heights too.
			// These late sessions might have their create claim window closed and are
			// no longer eligible to be claimed, but that's not always the case.
			// Once claim window closing is implemented, they will be filtered out
			// downstream at the waitForEarliestCreateClaimsHeight step.
			if claimWindowOpenHeight <= block.Height() {
				// Iterate over the suppliers' sessionsTrees that have grace period ending at
				// this block height and add them to the list of sessionTrees to be published.
				for _, supplierSessionTrees := range sessionsTreesEndingAtBlockHeight {
					// Skip the supplier if it has no sessions to claim.
					sessionTree, ok := supplierSessionTrees[sessionsSupplier]
					if !ok {
						continue
					}

					// Mark the session as claimed and add it to the list of sessionTrees to be published.
					// If the session has already been claimed, it will be skipped.
					// Appending the sessionTree to the list of sessionTrees is protected
					// against concurrent access by the sessionsTreesMu such that the first
					// call that marks the session as claimed will be the only one to add the
					// sessionTree to the list.
					if err := sessionTree.StartClaiming(); err != nil {
						continue
					}

					// Separate the sessions that are on-time from the ones that are late.
					// If the session is past its claim window open height, it is considered
					// late, otherwise it is on time and will be emitted last.
					if claimWindowOpenHeight < block.Height() {
						lateSessions = append(lateSessions, sessionTree)
					} else {
						onTimeSessions = append(onTimeSessions, sessionTree)
					}
				}

				// If there are any late sessions to be claimed, emit them first.
				// The wait for claim submission window pipeline step will return immediately
				// without blocking them.
				if len(lateSessions) > 0 {
					sessionsToClaimsPublishCh <- lateSessions
				}
			}
		}

		// Emit the on-time sessions last, after all the late sessions have been emitted.
		if len(onTimeSessions) > 0 {
			sessionsToClaimsPublishCh <- onTimeSessions
		}
	}
}

// removeFromRelayerSessions removes the SessionTree from the relayerSessions.
func (rs *relayerSessionsManager) removeFromRelayerSessions(sessionTree relayer.SessionTree) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	sessionHeader := sessionTree.GetSessionHeader()
	supplierOperatorAddress := sessionTree.GetSupplierOperatorAddress().String()

	logger := rs.logger.With("session_end_block_height", sessionHeader.SessionEndBlockHeight)

	sessionsTreesEndingAtBlockHeight, ok := rs.sessionsTrees[sessionHeader.SessionEndBlockHeight]
	if !ok {
		logger.Debug().Msg("no session trees found for the session end height")
		return
	}

	logger = logger.With("session_id", sessionHeader.SessionId)

	suppliersSessionTrees, ok := sessionsTreesEndingAtBlockHeight[sessionHeader.SessionId]
	if !ok {
		logger.Debug().Msg("no session trees found for the session id")
		return
	}

	logger = logger.With("supplier_operator_address", supplierOperatorAddress)

	_, ok = suppliersSessionTrees[supplierOperatorAddress]
	if !ok {
		logger.Debug().Msg("no session tree found for the supplier operator address")
		return
	}

	delete(suppliersSessionTrees, supplierOperatorAddress)

	// Check if the suppliersSessionTrees map is empty and delete it if so.
	if len(suppliersSessionTrees) == 0 {
		delete(sessionsTreesEndingAtBlockHeight, sessionHeader.SessionId)
	}

	// Check if the sessionsTreesEndingAtBlockHeight map is empty and delete it if so.
	// This is an optimization done to save memory by avoiding an endlessly growing sessionsTrees map.
	if len(sessionsTreesEndingAtBlockHeight) == 0 {
		delete(rs.sessionsTrees, sessionHeader.SessionEndBlockHeight)
	}
}

// validateConfig validates the relayerSessionsManager's configuration.
// TODO_TEST: Add unit tests to validate these configurations.
func (rs *relayerSessionsManager) validateConfig() error {
	if rs.storesDirectory == "" {
		return ErrSessionTreeUndefinedStoresDirectory
	}

	return nil
}

// waitForBlock blocks until the block at the given height (or greater) is
// observed as having been committed.
func (rs *relayerSessionsManager) waitForBlock(ctx context.Context, targetHeight int64) client.Block {
	// Create a cancellable child context for managing the CommittedBlocksSequence lifecycle.
	// Since the committedBlocksObserver is no longer needed after the block it is looking for
	// is reached, cancelling the child context at the end of the function will stop
	// the subscriptions and close the publish channel associated with the
	// CommittedBlocksSequence observable which is not exposing it.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	committedBlocksObs := rs.blockClient.CommittedBlocksSequence(ctx)
	committedBlocksObserver := committedBlocksObs.Subscribe(ctx)

	// minNumReplayBlocks is the number of blocks that MUST be in the block client's
	// replay buffer such that the target block can be observed.
	//
	// Plus one is necessary for the "oldest" boundary to include targetHeight.
	//
	// If minNumReplayBlocks is negative, no replay is necessary and the replay buffer will be ignored.
	minNumReplayBlocks := rs.blockClient.LastBlock(ctx).Height() - targetHeight + 1

	// TODO_MAINNET: If the replay buffer size is less than minNumReplayBlocks, the target
	// block targetHeight will never be observed. This can happen if a relayminer is cold-started
	// with persisted but unclaimed/unproven ("late") sessions, where a "late" session is one
	// which is unclaimed and whose earliest claim commit height has already elapsed.
	//
	// In this case, we should use a block query client to populate the block client replay
	// observable at the time of block client construction.
	// The latestBufferedOffset would be the difference between the current height and
	// earliest unclaimed/unproven session's earliest supplier claim/proof commit height.
	// This check and return branch can be removed once this is implemented.
	if committedBlocksObs.GetReplayBufferSize() < int(minNumReplayBlocks) {
		return nil
	}

	for block := range committedBlocksObserver.Ch() {
		if block.Height() >= targetHeight {
			return block
		}
	}

	return nil
}

// mapAddMinedRelayToSessionTree is intended to be used as a MapFn. It adds the relay
// to the session tree. If it encounters an error, it returns the error. Otherwise,
// it skips output (only outputs errors).
func (rs *relayerSessionsManager) mapAddMinedRelayToSessionTree(
	ctx context.Context,
	relay *relayer.MinedRelay,
) (_ error, skip bool) {
	// ensure the session tree exists for this relay
	// TODO_CONSIDERATION: if we get the session header from the response, there
	// is no possibility that we forgot to hydrate it (i.e. blindly trust the client).
	relayMetadata := relay.GetReq().GetMeta()
	smst, err := rs.ensureSessionTree(&relayMetadata)
	if err != nil {
		// TODO_IMPROVE: log additional info?
		rs.logger.Error().Err(err).Msg("failed to ensure session tree")
		return err, false
	}

	logger := rs.logger.
		With("session_id", smst.GetSessionHeader().GetSessionId()).
		With("application", smst.GetSessionHeader().GetApplicationAddress()).
		With("supplier_operator_address", smst.GetSupplierOperatorAddress().String())

	serviceComputeUnitsPerRelay, err := rs.getServiceComputeUnitsPerRelay(ctx, &relayMetadata)
	if err != nil {
		rs.logger.Error().Err(err).Msg("failed to get service compute units per relay")
		return err, false
	}

	// The weight of each relay is specified by the corresponding service's ComputeUnitsPerRelay field.
	// This is independent of the relay difficulty target hash for each service, which is supplied by the tokenomics module.
	if err := smst.Update(relay.Hash, relay.Bytes, serviceComputeUnitsPerRelay); err != nil {
		// TODO_IMPROVE: log additional info?
		logger.Error().Err(err).Msg("failed to update smt")
		return err, false
	}

	logger.Debug().Msg("added relay to session tree")

	// Skip because this map function only outputs errors.
	return nil, true
}

// deleteExpiredSessionTreesFn returns a function that deletes non-claimed sessions
// that have expired.
func (rs *relayerSessionsManager) deleteExpiredSessionTreesFn(
	expirationHeightFn func(*sharedtypes.Params, int64) int64,
) func(ctx context.Context, failedSessionTrees []relayer.SessionTree) {
	return func(ctx context.Context, failedSessionTrees []relayer.SessionTree) {
		currentHeight := rs.blockClient.LastBlock(ctx).Height()
		sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
		if err != nil {
			rs.logger.Error().Err(err).Msg("unable to query shared module params")
			return
		}

		for _, sessionTree := range failedSessionTrees {
			sessionEndHeight := sessionTree.GetSessionHeader().GetSessionEndBlockHeight()
			proofWindowCloseHeight := expirationHeightFn(sharedParams, sessionEndHeight)

			if currentHeight > proofWindowCloseHeight {
				rs.logger.Debug().Msg("deleting expired session")
				rs.removeFromRelayerSessions(sessionTree)
				if err := sessionTree.Delete(); err != nil {
					rs.logger.Error().
						Err(err).
						Str("session_id", sessionTree.GetSessionHeader().GetSessionId()).
						Str("supplier_operator_address", sessionTree.GetSupplierOperatorAddress().String()).
						Msg("failed to delete session tree")
				}
				continue
			}
		}
	}
}

// supplierSessionsToClaim returns an observable that notifies when sessions that
// are handled by the given supplier are ready to be claimed.
func (rs *relayerSessionsManager) supplierSessionsToClaim(
	ctx context.Context,
	supplierOperatorAddress string,
) observable.Observable[[]relayer.SessionTree] {
	sessionsToClaimObs, sessionsToClaimPublishCh := channel.NewObservable[[]relayer.SessionTree]()
	channel.ForEach(
		ctx,
		rs.blockClient.CommittedBlocksSequence(ctx),
		rs.forEachBlockClaimSessionsFn(supplierOperatorAddress, sessionsToClaimPublishCh),
	)

	return sessionsToClaimObs
}
