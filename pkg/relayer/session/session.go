package session

import (
	"context"
	"log"
	"sync"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ relayer.RelayerSessionsManager = (*relayerSessionsManager)(nil)

type sessionsTreesMap = map[int64]map[string]relayer.SessionTree

// relayerSessionsManager is an implementation of the RelayerSessions interface.
// TODO_TEST: Add tests to the relayerSessionsManager.
type relayerSessionsManager struct {
	relayObs relayer.MinedRelaysObservable

	// sessionsToClaimObs notifies about sessions that are ready to be claimed.
	sessionsToClaimObs observable.Observable[relayer.SessionTree]

	// sessionTrees is a map of block heights pointing to a map of SessionTrees
	// indexed by their sessionId.
	// The block height index is used to know when the sessions contained in the entry should be closed,
	// this helps to avoid iterating over all sessionsTrees to check if they are ready to be closed.
	sessionsTrees   sessionsTreesMap
	sessionsTreesMu *sync.Mutex

	// blockClient is used to get the notifications of committed blocks.
	blockClient client.BlockClient

	// supplierClient is used to create claims and submit proofs for sessions.
	supplierClient client.SupplierClient

	// storesDirectory points to a path on disk where KVStore data files are created.
	storesDirectory string
}

// NewRelayerSessions creates a new relayerSessions.
//
// Required dependencies:
//   - client.BlockClient
//   - client.SupplierClient
//
// Available options:
//   - WithStoresDirectory
func NewRelayerSessions(
	ctx context.Context,
	deps depinject.Config,
	opts ...relayer.RelayerSessionsManagerOption,
) (relayer.RelayerSessionsManager, error) {
	rs := &relayerSessionsManager{
		sessionsTrees:   make(sessionsTreesMap),
		sessionsTreesMu: &sync.Mutex{},
	}

	if err := depinject.Inject(
		deps,
		&rs.blockClient,
		&rs.supplierClient,
	); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(rs)
	}

	if err := rs.validateConfig(); err != nil {
		return nil, err
	}

	rs.sessionsToClaimObs = channel.MapExpand[client.Block, relayer.SessionTree](
		ctx,
		rs.blockClient.CommittedBlocksSequence(ctx),
		rs.mapBlockToSessionsToClaim,
	)

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

	// Start claim/proof pipeline.
	claimedSessionsObs := rs.createClaims(ctx)
	rs.submitProofs(ctx, claimedSessionsObs)
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

// ensureSessionTree returns the SessionTree for a given session.
// If no tree for the session exists, a new SessionTree is created before returning.
func (rs *relayerSessionsManager) ensureSessionTree(sessionHeader *sessiontypes.SessionHeader) (relayer.SessionTree, error) {
	sessionsTrees, ok := rs.sessionsTrees[sessionHeader.SessionEndBlockHeight]

	// If there is no map for sessions at the sessionEndHeight, create one.
	if !ok {
		sessionsTrees = make(map[string]relayer.SessionTree)
		rs.sessionsTrees[sessionHeader.SessionEndBlockHeight] = sessionsTrees
	}

	// Get the sessionTree for the given session.
	sessionTree, ok := sessionsTrees[sessionHeader.SessionId]

	// If the sessionTree does not exist, create it.
	var err error
	if !ok {
		sessionTree, err = NewSessionTree(sessionHeader, rs.storesDirectory, rs.removeFromRelayerSessions)
		if err != nil {
			return nil, err
		}

		sessionsTrees[sessionHeader.SessionId] = sessionTree
	}

	return sessionTree, nil
}

// mapBlockToSessionsToClaim maps a block to a list of sessions which can be
// claimed as of that block.
func (rs *relayerSessionsManager) mapBlockToSessionsToClaim(
	_ context.Context,
	block client.Block,
) (sessionTrees []relayer.SessionTree, skip bool) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	// Check if there are sessions that need to enter the claim/proof phase
	// as their end block height was the one before the last committed block.
	// Iterate over the sessionsTrees map to get the ones that end at a block height
	// lower than the current block height.
	for endBlockHeight, sessionsTreesEndingAtBlockHeight := range rs.sessionsTrees {
		// TODO_BLOCKER(@red-0ne): We need this to be == instead of <= because we don't want to keep sending
		// the same session while waiting the next step. This does not address the case
		// where the block client misses the target block which should be handled by the
		// retry mechanism. See the discussion in the following GitHub thread for next
		// steps: https://github.com/pokt-network/poktroll/pull/177/files?show-viewed-files=true&file-filters%5B%5D=#r1391957041
		if endBlockHeight == block.Height() {
			// Iterate over the sessionsTrees that end at this block height (or
			// less) and add them to the list of sessionTrees to be published.
			for _, sessionTree := range sessionsTreesEndingAtBlockHeight {
				sessionTrees = append(sessionTrees, sessionTree)
			}
		}
	}
	return sessionTrees, false
}

// removeFromRelayerSessions removes the SessionTree from the relayerSessions.
func (rs *relayerSessionsManager) removeFromRelayerSessions(sessionHeader *sessiontypes.SessionHeader) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	sessionsTreesEndingAtBlockHeight, ok := rs.sessionsTrees[sessionHeader.SessionEndBlockHeight]
	if !ok {
		log.Printf("DEBUG: no session tree found for sessions ending at height %d", sessionHeader.SessionEndBlockHeight)
		return
	}

	delete(sessionsTreesEndingAtBlockHeight, sessionHeader.SessionId)

	// Check if the sessionsTrees map is empty and delete it if so.
	// This is an optimization done to save memory by avoiding an endlessly growing sessionsTrees map.
	if len(sessionsTreesEndingAtBlockHeight) == 0 {
		delete(rs.sessionsTrees, sessionHeader.SessionEndBlockHeight)
	}
}

// validateConfig validates the relayerSessionsManager's configuration.
// TODO_TEST: Add unit tests to validate these configurations.
func (rp *relayerSessionsManager) validateConfig() error {
	if rp.storesDirectory == "" {
		return ErrSessionTreeUndefinedStoresDirectory
	}

	return nil
}

// waitForBlock blocks until the block at the given height (or greater) is
// observed as having been committed.
func (rs *relayerSessionsManager) waitForBlock(ctx context.Context, height int64) client.Block {
	subscription := rs.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx)
	defer subscription.Unsubscribe()

	for block := range subscription.Ch() {
		if block.Height() >= height {
			return block
		}
	}

	return nil
}

// mapAddMinedRelayToSessionTree is intended to be used as a MapFn. It adds the relay
// to the session tree. If it encounters an error, it returns the error. Otherwise,
// it skips output (only outputs errors).
func (rs *relayerSessionsManager) mapAddMinedRelayToSessionTree(
	_ context.Context,
	relay *relayer.MinedRelay,
) (_ error, skip bool) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()
	// ensure the session tree exists for this relay
	// TODO_CONSIDERATION: if we get the session header from the response, there
	// is no possibility that we forgot to hydrate it (i.e. blindly trust the client).
	sessionHeader := relay.GetReq().GetMeta().GetSessionHeader()
	smst, err := rs.ensureSessionTree(sessionHeader)
	if err != nil {
		log.Printf("ERROR: failed to ensure session tree: %s\n", err)
		return err, false
	}

	if err := smst.Update(relay.Hash, relay.Bytes, 1); err != nil {
		log.Printf("ERROR: failed to update smt: %s\n", err)
		return err, false
	}

	// Skip because this map function only outputs errors.
	return nil, true
}
