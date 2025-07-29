package session

import (
	"context"
	"path"
	"sync"
	"sync/atomic"

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
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Ensure the relayerSessionsManager implements the RelayerSessions interface.
var _ relayer.RelayerSessionsManager = (*relayerSessionsManager)(nil)

// SessionTreesMap is an alias type for a map of
// supplierOperatorAddress ->  sessionEndHeight -> sessionId -> SessionTree.
//
// It keeps track of the sessions created by RelayMiner in memory.
// The sessions are group by their end block height, session id and supplier operator address.
type SessionsTreesMap = map[string]map[int64]map[string]relayer.SessionTree

// relayerSessionsManager is an implementation of the RelayerSessions interface.
type relayerSessionsManager struct {
	logger polylog.Logger

	relayObs relayer.MinedRelaysObservable

	// TODO_TECHDEBT(@olshansk, @red-0ne):
	// 1. Review all usages of `sessionTrees` and simplify
	// 2. Ensure the mutex is used everywhere it's needed and is not used everywhere it's not
	// 3. Cleanup comments and techdebt in this package.
	//
	// sessionTrees is a SessionsTreesMap (see type alias above).
	//
	// - The block height index is used to know when the sessions contained in the entry should be closed.
	// - This helps to avoid iterating over all sessionsTrees every time when closing sessions.
	// - The sessionTrees are grouped by supplierOperatorAddress since each supplier has to claim its work done.
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

	// sessionSMTStore is a key-value store used to persist the metadata of
	// sessions created in order to recover the active ones in case of a restart.
	sessionSMTStore pebble.PebbleKVStore

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

	// stopping indicates whether the relayerSessionsManager is in the process of graceful shutdown.
	//
	// Why it exists:
	// - During normal operation, context cancellations (e.g., deadlines) are treated as session failures.
	// - These failures trigger cleanup: session trees are deleted.
	//
	// What changes when stopping = true:
	// - Context cancellations during shutdown are expected.
	// - These should NOT trigger deletion.
	// - This ensures session trees are persisted for recovery after restart.
	stopping atomic.Bool
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
	sessionSMTDir := path.Join(rs.storesDirectory, "sessions_metadata")
	if rs.sessionSMTStore, err = pebble.NewKVStore(sessionSMTDir); err != nil {
		return nil, err
	}

	return rs, nil
}

// Start maps over the session trees at the end of each respective session.
//
// The session trees are piped through a series of map operations which:
//   - Progress them through the claim/proof lifecycle
//   - Broadcast transactions to the network as necessary
//
// It IS NOT BLOCKING as map operations run in their own goroutines.
func (rs *relayerSessionsManager) Start(ctx context.Context) error {
	// Retrieve the latest block, which provides a reference height to:
	//   - Determine which sessions are still active
	//   - Identify which sessions have expired based on their end heights
	block := rs.blockClient.LastBlock(ctx)

	rs.logger.Info().Msgf(
		"📊 Chain head at height %d (block hash: %X) during session manager startup",
		block.Height(),
		block.Hash(),
	)

	// Restore previously active sessions from persistent storage by rehydrating
	// the session tree map.
	// This is crucial for:
	//   - Preserving the relayer's state across restarts
	//   - Ensuring no active sessions are lost when the process is interrupted
	//   - Maintaining accumulated work when interruptions occur
	if err := rs.loadSessionTreeMap(ctx, block.Height()); err != nil {
		return err
	}

	// DEV_NOTE: must cast back to generic observable type to use with Map.
	// relayer.MinedRelaysObservable cannot be an alias due to gomock's lack of
	// support for generic types.
	relayObs := observable.Observable[*relayer.MinedRelay](rs.relayObs)

	// Map eitherMinedRelays to a new observable of an error type which is
	// notified if an error occurs when attempting to add the relay to the session tree.
	miningErrorsObs := channel.Map(ctx, relayObs, rs.mapAddMinedRelayToSessionTree)
	logging.LogErrors(ctx, miningErrorsObs)

	// Start claim/proof pipeline for each supplier that is present in the RelayMiner.
	for supplierOperatorAddress, supplierClient := range rs.supplierClients.SupplierClients {
		supplierSessionsToClaimObs := rs.supplierSessionsToClaim(ctx, supplierOperatorAddress)
		claimedSessionsObs := rs.createClaims(ctx, supplierClient, supplierSessionsToClaimObs)
		rs.submitProofs(ctx, supplierClient, claimedSessionsObs)
	}

	// Stop the relayer sessions manager when the context is done.
	// This is necessary to ensure that during shutdown:
	//   - All session trees are persisted
	//   - Their root hashes are preserved
	go func() {
		<-ctx.Done()
		rs.Stop()
	}()

	return nil
}

// Stop performs a complete shutdown of the relayerSessionsManager by:
//   - Closing connections and canceling subscriptions
//   - Persisting all session data to storage
//   - Releasing resources and clearing memory
//
// This ensures no data is lost during shutdown and resources are properly cleaned up.
func (rs *relayerSessionsManager) Stop() {
	// Mark the manager as stopping to prevent misinterpreting shutdown cancellations as failures.
	//
	// This ensures:
	// - Session trees are not deleted during shutdown.
	// - Data is preserved for recovery on the next startup.
	rs.stopping.Store(true)

	// Close the block client and unsubscribe from all observables to stop receiving events.
	// Proper shutdown is important for:
	//   - Graceful termination
	//   - Testing scenarios
	// While process termination would eventually clean these up, explicit cleanup is preferred.
	rs.blockClient.Close()
	rs.relayObs.UnsubscribeAll()

	// Lock the mutex before accessing and modifying the sessionsTrees map to ensure
	// thread safety during shutdown.
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	// Persist each active session's state to disk and properly close the associated
	// key-value stores. This ensures that all accumulated relay data (including root
	// hashes needed for claims) is safely stored before shutdown.
	numSessionTrees := 0
	for _, supplierSessionTrees := range rs.sessionsTrees {
		for _, sessionTreesAtHeight := range supplierSessionTrees {
			for _, sessionTree := range sessionTreesAtHeight {
				sessionId := sessionTree.GetSessionHeader().GetSessionId()

				logger := rs.logger.
					With("method", "RSM.Stop").
					With("session_id", sessionId).
					With("supplier_operator_address", sessionTree.GetSupplierOperatorAddress())

				// Store the session tee to disk
				if err := rs.persistSessionMetadata(sessionTree); err != nil {
					logger.Error().Err(err).Msg("❌️ Failed to persist session metadata to storage during shutdown. ❗Check disk space and permissions. ❗Session data may be lost on restart.")
				}

				// Stop the session tree process and underlying key-value store.
				if err := sessionTree.Stop(); err != nil {
					logger.Error().Err(err).Msg("❌️ Failed to stop session tree store during shutdown. ❗Check disk permissions and kvstore integrity. ❗Resources may not be properly cleaned up.")
				}

				logger.Debug().Msg("💾 Successfully stored session tree to disk during shutdown")
				numSessionTrees++
			}
		}
	}

	// Close the metadata store that tracks all sessions and release its resources.
	if err := rs.sessionSMTStore.Stop(); err != nil {
		rs.logger.Error().Err(err).Msg("❌️ Failed to stop sessions metadata store during shutdown. ❗Check disk permissions and kvstore integrity. ❗Resources may not be properly cleaned up.")
	}

	clear(rs.sessionsTrees)
	rs.logger.Info().Msgf("🧹 Successfully cleared %d session trees from memory during shutdown", numSessionTrees)
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
func (rs *relayerSessionsManager) forEachBlockClaimSessionsFn(
	sessionsSupplier string,
	sessionsToClaimsPublishCh chan<- []relayer.SessionTree,
) channel.ForEachFn[client.Block] {
	return func(ctx context.Context, block client.Block) {
		// onTimeSessions are the sessions that are still within their grace period.
		// They are on time and will wait for their create claim window to open.
		// They will be emitted last, after all the late sessions have been emitted.
		var onTimeSessions []relayer.SessionTree

		// Query shared params before acquiring the mutex to avoid blocking session tree operations
		sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
		if err != nil {
			rs.logger.Error().Err(err).Msg("❌️ Failed to query shared module parameters. ❗Check node connectivity and sync status. ❗Cannot process session claims without network parameters.")
			return
		}

		// Only hold the mutex while copying the data, not during processing
		rs.sessionsTreesMu.Lock()
		// Get the sessions trees for the supplier matching the sessionsSupplier.
		supplierSessionTress := rs.sessionsTrees[sessionsSupplier]

		// Collect sessions that need to be processed while holding the mutex
		// Map from sessionEndHeight to list of sessions to process
		sessionsByEndHeight := make(map[int64][]relayer.SessionTree)
		for sessionEndHeight, sessionsTreesEndingAtBlockHeight := range supplierSessionTress {
			claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)
			// Only collect sessions that are ready to be claimed
			if claimWindowOpenHeight <= block.Height() {
				// Create a copy of sessions to process later
				var sessionsCopy []relayer.SessionTree
				for _, sessionTree := range sessionsTreesEndingAtBlockHeight {
					// Mark the session as claimed while holding the mutex
					if err := sessionTree.StartClaiming(); err != nil {
						continue
					}
					sessionsCopy = append(sessionsCopy, sessionTree)
				}
				if len(sessionsCopy) > 0 {
					sessionsByEndHeight[sessionEndHeight] = sessionsCopy
				}
			}
		}
		rs.sessionsTreesMu.Unlock()
		// Mutex is now released - session tree operations can proceed

		// Process the collected sessions without holding the mutex
		// This allows new relays to be added to session trees while claims are being processed
		var lateSessions [][]relayer.SessionTree
		for sessionEndHeight, sessions := range sessionsByEndHeight {
			claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)
			var lateSessionsBatch []relayer.SessionTree
			for _, sessionTree := range sessions {
				// Separate the sessions that are on-time from the ones that are late.
				// If the session is past its claim window open height, it is considered
				// late, otherwise it is on time and will be emitted last.
				if claimWindowOpenHeight < block.Height() {
					lateSessionsBatch = append(lateSessionsBatch, sessionTree)
				} else {
					onTimeSessions = append(onTimeSessions, sessionTree)
				}
			}

			// Collect late sessions by batch to emit them first
			if len(lateSessionsBatch) > 0 {
				lateSessions = append(lateSessions, lateSessionsBatch)
			}
		}

		// Emit late sessions first (they need immediate processing)
		for _, lateBatch := range lateSessions {
			sessionsToClaimsPublishCh <- lateBatch
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

	logger := rs.logger.
		With("method", "RSM.removeFromRelayerSessions").
		With("supplier_operator_address", supplierOperatorAddress)

	supplierSessionTrees, ok := rs.sessionsTrees[supplierOperatorAddress]
	if !ok {
		logger.Debug().Msg("🔍 No session trees found for supplier operator address - skipping removal")
		return
	}

	logger = logger.With("session_end_block_height", sessionHeader.SessionEndBlockHeight)

	sessionsTreesEndingAtBlockHeight, ok := supplierSessionTrees[sessionHeader.SessionEndBlockHeight]
	if !ok {
		logger.Debug().Msg("🔍 No session trees found for session end height - skipping removal")
		return
	}

	logger = logger.With("session_id", sessionHeader.SessionId)

	_, ok = sessionsTreesEndingAtBlockHeight[sessionHeader.SessionId]
	if !ok {
		logger.Debug().Msg("🔍 No session tree found for session ID - already removed or never existed")
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
	currentBlock := rs.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()
	minNumReplayBlocks := currentHeight - targetHeight + 1

	rs.logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) while waiting for target block %d",
		currentHeight,
		currentBlock.Hash(),
		targetHeight,
	)

	// If the replay buffer size is less than minNumReplayBlocks, the target
	// block targetHeight will never be observed. This can happen if a relayminer is cold-started
	// with persisted but unclaimed/unproven ("late") sessions, where a "late" session is one
	// which is unclaimed and whose earliest claim commit height has already elapsed.
	if committedBlocksObs.GetReplayBufferSize() < int(minNumReplayBlocks) {
		blockResult, err := rs.blockQueryClient.Block(ctx, &targetHeight)
		if err != nil {
			rs.logger.Error().Err(err).Msgf("❌️ Failed to query block at height %d. ❗Check node connectivity and sync status. ❗Session timing calculations may be affected.", targetHeight)
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

	logger := rs.logger.
		With("session_id", relayMetadata.GetSessionHeader().GetSessionId()).
		With("application", relayMetadata.GetSessionHeader().GetApplicationAddress()).
		With("supplier_operator_address", relayMetadata.GetSupplierOperatorAddress())

	smst, err := rs.ensureSessionTree(&relayMetadata)
	if err != nil {
		// TODO_IMPROVE: log additional info?
		logger.Error().Err(err).Msg("❌️ Failed to ensure session tree exists for relay. ❗Check disk space and kvstore integrity. ❗Relay cannot be processed.")
		return err, false
	}

	serviceComputeUnitsPerRelay, err := rs.getServiceComputeUnitsPerRelay(ctx, &relayMetadata)
	if err != nil {
		logger.Error().Err(err).Msg("❌️ Failed to get service compute units per relay. ❗Check service configuration and node connectivity. ❗Relay weight calculation cannot proceed.")
		return err, false
	}

	// The weight of each relay is specified by the corresponding service's ComputeUnitsPerRelay field.
	// This is independent of the relay difficulty target hash for each service, which is supplied by the tokenomics module.
	if err := smst.Update(relay.Hash, relay.Bytes, serviceComputeUnitsPerRelay); err != nil {
		// TODO_IMPROVE: log additional info?
		logger.Error().Err(err).Msg("❌️ Failed to update session merkle tree with relay data. ❗Check disk space and kvstore integrity. ❗Relay evidence may be lost.")
		return err, false
	}

	logger.Debug().Msg("⛏️ Successfully added relay to session tree for claim accumulation")

	// Skip because this map function only outputs errors.
	return nil, true
}

// deleteExpiredSessionTreesFn deletes unclaimed sessions past the proof window close height.
// These sessions can no longer be proved onchain, so there is no need for the offchain evidence (i.e. the session tree).
func (rs *relayerSessionsManager) deleteExpiredSessionTreesFn(
	supplierOperatorAddress string,
) func(ctx context.Context, currentBlock client.Block) {
	return func(ctx context.Context, currentHeight client.Block) {
		logger := rs.logger.
			With("method", "RSM.deleteExpiredSessionTreesFn").
			With("supplier_operator_address", supplierOperatorAddress)

		sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("❌️ Failed to query shared module parameters for session expiry check. ❗Check node connectivity and sync status. ❗Cannot determine session expiration timing.")
			return
		}

		// Lock mutex to safely read from the sessionsTrees map
		rs.sessionsTreesMu.Lock()

		supplierSessionTrees, ok := rs.sessionsTrees[supplierOperatorAddress]
		if !ok || supplierSessionTrees == nil {
			rs.sessionsTreesMu.Unlock() // Unlock before returning
			// Use probabilistic debug info to log that no session trees were found to avoid spamming
			// the logs with entries at each new block height and supplier that has no session trees.
			logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).
				Msg("🔍 No expired session trees found for supplier operator address - all sessions still active")
			return
		}

		// Create a copy of the relevant trees to avoid holding the lock
		// during the potentially time-consuming operations that follow
		expiredSessionTrees := make([]relayer.SessionTree, 0)
		for _, sessionTrees := range supplierSessionTrees {
			for sessionId, sessionTree := range sessionTrees {
				sessionHeader := sessionTree.GetSessionHeader()
				sessionEndHeight := sessionHeader.GetSessionEndBlockHeight()
				proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(sharedParams, sessionEndHeight)
				currentHeight := currentHeight.Height()
				// If the session is already past its proof window close height,
				// it is considered expired and should be deleted.
				if currentHeight > proofWindowCloseHeight {
					logger.Info().
						Str("service_id", sessionHeader.GetServiceId()).
						Str("application_address", sessionHeader.GetApplicationAddress()).
						Str("session_id", sessionId).
						Msgf("🗑️ Marking expired session for deletion - proof window closed at height %d (current: %d). Session can no longer earn rewards.",
							proofWindowCloseHeight, currentHeight)

					expiredSessionTrees = append(expiredSessionTrees, sessionTree)
				}
			}
		}

		// Unlock the mutex after we're done reading the map
		rs.sessionsTreesMu.Unlock()

		// Delete the expired session trees from the relayerSessions.
		rs.deleteSessionTrees(ctx, expiredSessionTrees)
	}
}

// deleteSessionTrees deletes the provided session trees from the relayerSessions.
// It removes the session tree from the in-memory map and deletes it from the disk store.
func (rs *relayerSessionsManager) deleteSessionTrees(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
) {
	logger := rs.logger.With("method", "RSM.deleteSessionTrees")

	if len(sessionTrees) == 0 {
		logger.Debug().Msg("🔍 No session trees to delete - deletion request was empty")
		return
	}

	logger = logger.With("supplier_operator_address", sessionTrees[0].GetSupplierOperatorAddress())

	// Iterate over the session trees and delete them from the relayerSessions.
	numSessionTreesDeleted := 0
	for _, sessionTree := range sessionTrees {
		sessionId := sessionTree.GetSessionHeader().GetSessionId()
		// Create a new logger instance for each iteration to avoid concurrent access issues
		sessionLogger := logger.With("session_id", sessionId)
		sessionLogger.Info().Msg("🗑️ Deleting session tree - cleaning up outdated or unclaimable session")

		// Remove the session tree from the relayerSessions.
		rs.deleteSessionTree(sessionTree)

		numSessionTreesDeleted++
	}

	logger.Debug().Msgf(
		"🧹 Successfully deleted %d session trees from memory and storage",
		numSessionTreesDeleted,
	)
}

// deleteSessionTree deletes the session tree from the relayerSessions and
// removes it from the disk store.
func (rs *relayerSessionsManager) deleteSessionTree(sessionTree relayer.SessionTree) {
	rs.removeFromRelayerSessions(sessionTree)

	sessionHeader := sessionTree.GetSessionHeader()
	logger := rs.logger.With(
		"session_id", sessionHeader.GetSessionId(),
		"application_address", sessionHeader.GetApplicationAddress(),
		"service_id", sessionHeader.GetServiceId(),
		"supplier_operator_address", sessionTree.GetSupplierOperatorAddress(),
	)

	// IMPORTANT: Create sessionSMT BEFORE deleting the tree
	// This ensures we retrieve the SMT root while the KVStore is still open.
	sessionSMT := sessionSMTFromSessionTree(sessionTree)

	// Delete the session tree from the KVStore and close the underlying store.
	if err := sessionTree.Delete(); err != nil {
		logger.Error().Err(err).Msg("❌️ Failed to delete session tree from kvstore. ❗Check disk permissions and kvstore integrity. ❗Session data may persist incorrectly.")
	}

	// Delete the persisted session tree metadata from the disk store.
	// This is necessary to ensure that the session is not restored on the next startup.
	if err := rs.deletePersistedSessionTree(sessionSMT); err != nil {
		logger.Error().
			Err(err).
			Msg("❌️ Failed to delete persisted session tree metadata from storage. ❗Check disk permissions and kvstore integrity. ❗Session may be restored on next startup.")
	}

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).
		Msg("🧹 Successfully deleted session tree from memory and disk storage - cleanup complete")
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

	// At each new block, check and clean up any expired sessions that are past
	// their proof window close height.
	channel.ForEach(
		ctx,
		rs.blockClient.CommittedBlocksSequence(ctx),
		rs.deleteExpiredSessionTreesFn(supplierOperatorAddress),
	)

	return sessionsToClaimObs
}

// claimFromSessionTree returns a claim object from the given SessionTree.
func claimFromSessionTree(sessionTree relayer.SessionTree) prooftypes.Claim {
	return prooftypes.Claim{
		SupplierOperatorAddress: sessionTree.GetSupplierOperatorAddress(),
		SessionHeader:           sessionTree.GetSessionHeader(),
		RootHash:                sessionTree.GetClaimRoot(),
	}
}

// sessionSMTFromSessionTree creates a SessionSMT from the given SessionTree.
func sessionSMTFromSessionTree(
	sessionTree relayer.SessionTree,
) *prooftypes.SessionSMT {
	return &prooftypes.SessionSMT{
		SessionHeader:           sessionTree.GetSessionHeader(),
		SupplierOperatorAddress: sessionTree.GetSupplierOperatorAddress(),
		SmtRoot:                 sessionTree.GetSMSTRoot(),
	}
}
