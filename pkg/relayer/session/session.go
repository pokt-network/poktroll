package session

import (
	"context"
	"log"
	"sync"

	blockclient "github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ relayer.RelayerSessionsManager = (*relayerSessionsManager)(nil)

type sessionsTreesMap = map[int64]map[string]relayer.SessionTree

// relayerSessionsManager is an implementation of the RelayerSessions interface.
// TODO_TEST: Add tests to the relayerSessionsManager.
type relayerSessionsManager struct {
	// sessionsToClaim notifies about sessions that are ready to be claimed.
	sessionsToClaim observable.Observable[relayer.SessionTree]

	// sessionsToClaimPublisher is the channel used to publish sessions to claim.
	sessionsToClaimPublisher chan<- relayer.SessionTree

	// sessionTrees is a map of block heights pointing to a map of SessionTrees
	// indexed by their sessionId.
	// The block height index is used to know when the sessions contained in the entry should be closed,
	// this helps to avoid iterating over all sessionsTrees to check if they are ready to be closed.
	sessionsTrees   sessionsTreesMap
	sessionsTreesMu *sync.Mutex

	// blockClient is used to get the notifications of committed blocks.
	blockClient blockclient.BlockClient

	// storesDirectory points to a path on disk where KVStore data files are created.
	storesDirectory string
}

// NewRelayerSessions creates a new relayerSessions.
func NewRelayerSessions(
	ctx context.Context,
	storesDirectory string,
	blockClient blockclient.BlockClient,
) relayer.RelayerSessionsManager {
	rs := &relayerSessionsManager{
		sessionsTrees:   make(sessionsTreesMap),
		storesDirectory: storesDirectory,
		blockClient:     blockClient,
	}
	rs.sessionsToClaim, rs.sessionsToClaimPublisher = channel.NewObservable[relayer.SessionTree]()

	go rs.goListenToCommittedBlocks(ctx)

	return rs
}

// SessionsToClaim returns an observable that notifies when sessions are ready to be claimed.
func (rs *relayerSessionsManager) SessionsToClaim() observable.Observable[relayer.SessionTree] {
	return rs.sessionsToClaim
}

// EnsureSessionTree returns the SessionTree for a given session.
// If no tree for the session exists, a new SessionTree is created before returning.
func (rs *relayerSessionsManager) EnsureSessionTree(sessionHeader *sessiontypes.SessionHeader) (relayer.SessionTree, error) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	sessionsTrees, ok := rs.sessionsTrees[sessionHeader.SessionEndBlockHeight]

	// If there is no map for sessions at the sessionEndHeight, create one.
	if !ok {
		sessionsTrees = make(map[string]relayer.SessionTree)
		rs.sessionsTrees[sessionHeader.SessionEndBlockHeight] = sessionsTrees
	}

	// Get the sessionTree for the given session.
	sessionTree, ok := sessionsTrees[sessionHeader.SessionId]

	// If the sessionTree does not exist, create it.
	if !ok {
		sessionTree, err := NewSessionTree(sessionHeader, rs.storesDirectory, rs.removeFromRelayerSessions)
		if err != nil {
			return nil, err
		}

		sessionsTrees[sessionHeader.SessionId] = sessionTree
	}

	return sessionTree, nil
}

// goListenToCommittedBlocks listens to committed blocks so that rs.sessionsToClaimPublisher
// can notify when sessions are ready to be claimed.
// It is intended to be called as a background goroutine.
func (rs *relayerSessionsManager) goListenToCommittedBlocks(ctx context.Context) {
	committedBlocks := rs.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx).Ch()

	for block := range committedBlocks {
		// Check if there are sessions that need to enter the claim/proof phase
		// as their end block height was the one before the last committed block.
		// Iterate over the sessionsTrees map to get the ones that end at a block height
		// lower than the current block height.
		for endBlockHeight, sessionsTreesEndingAtBlockHeight := range rs.sessionsTrees {
			if endBlockHeight < block.Height() {
				// Iterate over the sessionsTrees that end at this block height and publish them.
				for _, sessionTree := range sessionsTreesEndingAtBlockHeight {
					rs.sessionsToClaimPublisher <- sessionTree
				}
			}
		}
	}
}

// removeFromRelayerSessions removes the SessionTree from the relayerSessions.
func (rs *relayerSessionsManager) removeFromRelayerSessions(sessionHeader *sessiontypes.SessionHeader) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	sessionsTreesEndingAtBlockHeight, ok := rs.sessionsTrees[sessionHeader.SessionEndBlockHeight]
	if !ok {
		log.Printf("no session tree found for sessions ending at height %d", sessionHeader.SessionEndBlockHeight)
		return
	}

	delete(sessionsTreesEndingAtBlockHeight, sessionHeader.SessionId)

	// Check if the sessionsTrees map is empty and delete it if so.
	// This is an optimization done to save memory by avoiding an endlessly growing sessionsTrees map.
	if len(sessionsTreesEndingAtBlockHeight) == 0 {
		delete(rs.sessionsTrees, sessionHeader.SessionEndBlockHeight)
	}
}
