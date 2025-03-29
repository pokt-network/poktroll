package session

import (
	"context"
	"path"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/smt/kvstore/pebble"

	"github.com/pokt-network/poktroll/pkg/client"
	blocktypes "github.com/pokt-network/poktroll/pkg/client/block"
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

// SessionTreesMap is an alias type for a map of
// supplierOperatorAddress->sessionEndHeight->sessionId->SessionTree.
// It is used to keep track of the sessions that are created in the RelayMiner
// by grouping them by their end block height, session id and supplier operator address.
type SessionsTreesMap = map[string]map[int64]map[string]relayer.SessionTree

// relayerSessionsManager is an implementation of the RelayerSessions interface.
// TODO_TEST: Add tests to the relayerSessionsManager.
type relayerSessionsManager struct {
	logger polylog.Logger

	relayObs relayer.MinedRelaysObservable

	// sessionTrees is a map of supplierOperatorAddress->blockHeight->sessionId->sessionTree.
	// The block height index is used to know when the sessions contained in the
	// entry should be closed, this helps to avoid iterating over all sessionsTrees
	// to check if they are ready to be closed.
	// The sessionTrees are grouped by supplierOperatorAddress since each supplier has to
	// claim the work it has been assigned.
	sessionsTrees   SessionsTreesMap
	sessionsTreesMu *sync.Mutex

	// blockClient is used to get the notifications of committed blocks.
	blockClient client.BlockClient

	// blockQueryClient is used to query for blocks by height in case the blockClient
	// does not have the block in its replay buffer.
	blockQueryClient client.BlockQueryClient

	// supplierClients is used to create claims and submit proofs for sessions.
	supplierClients *supplier.SupplierClientMap

	// storesDirectory points to a path on disk where KVStore data files are created.
	storesDirectory string

	// sessionsMetadataStore is a key-value store used to persist the metadata of
	// sessions created in order to recover the active ones in case of a restart.
	sessionsMetadataStore pebble.PebbleKVStore

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
//   - client.BlockQueryClient
//   - client.SupplierClientMap
//   - client.SharedQueryClient
//   - client.ServiceQueryClient
//   - client.ProofQueryClient
//   - client.BankQueryClient
//   - polylog.Logger
//
// Available options:
//   - WithStoresDirectory
//   - WithSigningKeyNames
func NewRelayerSessions(
	deps depinject.Config,
	opts ...relayer.RelayerSessionsManagerOption,
) (_ relayer.RelayerSessionsManager, err error) {
	rs := &relayerSessionsManager{
		sessionsTrees:   make(SessionsTreesMap),
		sessionsTreesMu: &sync.Mutex{},
	}

	if err = depinject.Inject(
		deps,
		&rs.blockClient,
		&rs.blockQueryClient,
		&rs.supplierClients,
		&rs.sharedQueryClient,
		&rs.serviceQueryClient,
		&rs.proofQueryClient,
		&rs.bankQueryClient,
		&rs.logger,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(rs)
	}

	if err = rs.validateConfig(); err != nil {
		return nil, err
	}

	// Initialize the session metadata store.
	sessionRegistryDir := path.Join(rs.storesDirectory, "sessions_metadata")
	if rs.sessionsMetadataStore, err = pebble.NewKVStore(sessionRegistryDir); err != nil {
		return nil, err
	}

	return rs, nil
}

// Start maps over the session trees at the end of each, respective, session.
// The session trees are piped through a series of map operations which progress
// them through the claim/proof lifecycle, broadcasting transactions to  the
// network as necessary.
// It IS NOT BLOCKING as map operations run in their own goroutines.
func (rs *relayerSessionsManager) Start(ctx context.Context) error {
	// Retrieve the latest block, its height will be used as a reference point to determine
	// which sessions are still active and which ones have expired based on their end heights.
	block := rs.blockClient.LastBlock(ctx)

	// Restore previously active sessions from persistent storage by rehydrating
	// the session tree map. This is crucial for preserving the relayer's state across
	// restarts, ensuring that no active sessions or accumulated work is lost when the
	// process is interrupted.
	if err := rs.populateSessionTreeMap(ctx, block.Height()); err != nil {
		return err
	}

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

	// Stop the relayer sessions manager when the context is done.
	// This is necessary to ensure that in case of a shutdown, all the sessions trees
	// are persisted along their root hashes.
	go func() {
		<-ctx.Done()
		rs.Stop()
	}()

	return nil
}

// Stop performs a complete shutdown of the relayerSessionsManager by:
// 1. Closing connections and canceling subscriptions
// 2. Persisting all session data to storage
// 3. Releasing resources and clearing memory
// This ensures no data is lost during shutdown and resources are properly cleaned up.
func (rs *relayerSessionsManager) Stop() {
	// Close the block client and unsubscribe from all observables to stop receiving events.
	// While process termination would eventually clean these up, proper shutdown
	// is important for graceful termination and testing scenarios.
	rs.blockClient.Close()
	rs.relayObs.UnsubscribeAll()

	// Persist each active session's state to disk and properly close the associated
	// key-value stores. This ensures that all accumulated relay data (including root
	// hashes needed for claims) is safely stored before shutdown.
	for _, supplierSessionTrees := range rs.sessionsTrees {
		for _, sessionTreesAtHeight := range supplierSessionTrees {
			for _, sessionTree := range sessionTreesAtHeight {
				if err := rs.persistSessionMetadata(sessionTree); err != nil {
					rs.logger.Error().Err(err).Msgf(
						"failed to persist session metadata for sessionId %q",
						sessionTree.GetSessionHeader().GetSessionId(),
					)
				}

				if err := sessionTree.Stop(); err != nil {
					rs.logger.Error().Err(err).Msgf(
						"failed to stop session tree store for sessionId %q",
						sessionTree.GetSessionHeader().GetSessionId(),
					)
				}
			}
		}
	}

	// Close the metadata store that tracks all sessions and release its resources.
	// Then clear the in-memory sessions map for testing purposes.
	if err := rs.sessionsMetadataStore.Stop(); err != nil {
		rs.logger.Error().Err(err).Msg("failed to stop sessions metadata store")
	}
	clear(rs.sessionsTrees)
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

	// Get the supplier session trees for the supplierOperatorAddress.
	supplierOperatorAddress := relayRequestMetadata.SupplierOperatorAddress
	supplierSessionTrees, ok := rs.sessionsTrees[supplierOperatorAddress]

	// If there is no map for session trees with the supplier operator address, create one.
	if !ok {
		supplierSessionTrees = make(map[int64]map[string]relayer.SessionTree)
		rs.sessionsTrees[supplierOperatorAddress] = supplierSessionTrees
	}

	// Get the sessions that end at the relay request's sessionEndHeight.
	sessionHeader := relayRequestMetadata.SessionHeader
	sessionTreesWithEndHeight, ok := supplierSessionTrees[sessionHeader.SessionEndBlockHeight]

	// If there is no map for sessions with sessionEndHeight, create one.
	if !ok {
		sessionTreesWithEndHeight = make(map[string]relayer.SessionTree)
		supplierSessionTrees[sessionHeader.SessionEndBlockHeight] = sessionTreesWithEndHeight
	}

	sessionTree, ok := sessionTreesWithEndHeight[sessionHeader.SessionId]

	// If the sessionTree does not exist, create and assign it to the
	// sessionTreeWithSessionId map for the given supplier operator address.
	if !ok {
		var err error
		sessionTree, err = NewSessionTree(sessionHeader, supplierOperatorAddress, rs.storesDirectory, rs.logger)
		if err != nil {
			return nil, err
		}

		sessionTreesWithEndHeight[sessionHeader.SessionId] = sessionTree

		// Persist the newly created session tree metadata to disk.
		if err := rs.persistSessionMetadata(sessionTree); err != nil {
			return nil, err
		}
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

		// Get the sessions trees for the supplier matching the sessionsSupplier.
		supplierSessionTress := rs.sessionsTrees[sessionsSupplier]

		// Check if there are sessions that need to enter the claim/proof phase as their
		// end block height was the one before the last committed block or earlier.
		// Iterate over the supplier sessionsTrees map to get the ones that end at a
		// block height lower than the current block height.
		for sessionEndHeight, sessionsTreesEndingAtBlockHeight := range supplierSessionTress {
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
				// Iterate over the sessionsTrees that have grace period ending at this
				// block height and add them to the list of sessionTrees to be published.
				for _, sessionTree := range sessionsTreesEndingAtBlockHeight {
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
	supplierOperatorAddress := sessionTree.GetSupplierOperatorAddress()

	logger := rs.logger.With("supplier_operator_address", supplierOperatorAddress)

	supplierSessionTrees, ok := rs.sessionsTrees[supplierOperatorAddress]
	if !ok {
		logger.Debug().Msg("no session tree found for the supplier operator address")
		return
	}

	logger = logger.With("session_end_block_height", sessionHeader.SessionEndBlockHeight)

	sessionsTreesEndingAtBlockHeight, ok := supplierSessionTrees[sessionHeader.SessionEndBlockHeight]
	if !ok {
		logger.Debug().Msg("no session trees found for the session end height")
		return
	}

	logger = logger.With("session_id", sessionHeader.SessionId)

	_, ok = sessionsTreesEndingAtBlockHeight[sessionHeader.SessionId]
	if !ok {
		logger.Debug().Msg("no session trees found for the session id")
		return
	}

	delete(sessionsTreesEndingAtBlockHeight, sessionHeader.SessionId)

	// Check if the suppliersSessionTrees map is empty and delete it if so.
	if len(sessionsTreesEndingAtBlockHeight) == 0 {
		delete(supplierSessionTrees, sessionHeader.SessionEndBlockHeight)
	}

	// Check if the sessionsTreesEndingAtBlockHeight map is empty and delete it if so.
	// This is an optimization done to save memory by avoiding an endlessly growing sessionsTrees map.
	if len(supplierSessionTrees) == 0 {
		delete(rs.sessionsTrees, supplierOperatorAddress)
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
	// is reached, canceling the child context at the end of the function will stop
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

	// If the replay buffer size is less than minNumReplayBlocks, the target
	// block targetHeight will never be observed. This can happen if a relayminer is cold-started
	// with persisted but unclaimed/unproven ("late") sessions, where a "late" session is one
	// which is unclaimed and whose earliest claim commit height has already elapsed.
	if committedBlocksObs.GetReplayBufferSize() < int(minNumReplayBlocks) {
		blockResult, err := rs.blockQueryClient.Block(ctx, &targetHeight)
		if err != nil {
			rs.logger.Error().Err(err).Msgf("failed to query for block block height %d", targetHeight)
			return nil
		}

		block := blocktypes.CometBlockResult(*blockResult)
		return &block
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
		With("supplier_operator_address", smst.GetSupplierOperatorAddress())

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

		// TODO_TEST: Add tests that cover existing expired failed session trees.
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
						Str("supplier_operator_address", sessionTree.GetSupplierOperatorAddress()).
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
