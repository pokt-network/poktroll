package session

import (
	"context"
	"log"
	blockclient "pocket/pkg/client"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
	sessiontypes "pocket/x/session/types"
	"sync"
)

var _ RelayerSessionsManager = (*relayerSessionsManager)(nil)

type (
	sessionId        = string
	blockHeight      = int64
	sessionsTreesMap = map[blockHeight]map[sessionId]SessionTree
)

// relayerSessionsManager is an implementation of the RelayerSessions interface.
type relayerSessionsManager struct {
	// closingSessions notifies about sessions that are ready to be claimed.
	closingSessions observable.Observable[SessionTree]

	// closingSessionsPublisher is the channel used to publish closing sessions.
	closingSessionsPublisher chan<- SessionTree

	// sessionTrees is a map of block heights containing another map of SessionTrees indexed by their sessionId.
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
) RelayerSessionsManager {
	rs := &relayerSessionsManager{
		sessionsTrees:   make(sessionsTreesMap),
		storesDirectory: storesDirectory,
		blockClient:     blockClient,
	}
	rs.closingSessions, rs.closingSessionsPublisher = channel.NewObservable[SessionTree]()

	go rs.goListenToCommittedBlocks(ctx)

	return rs
}

// ClosingSessions returns an observable that notifies when sessions are ready to be claimed.
func (rs *relayerSessionsManager) ClosingSessions() observable.Observable[SessionTree] {
	return rs.closingSessions
}

// EnsureSessionTree returns the SessionTree for a given session.
// If no tree for the session exists, a new SessionTree is created before returning.
func (rs *relayerSessionsManager) EnsureSessionTree(session *sessiontypes.Session) (SessionTree, error) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	// Calculate the session end height based on the session start block height
	// and the number of blocks per session.
	sessionEndHeight := session.Header.SessionStartBlockHeight + session.NumBlocksPerSession
	sessionsTrees, ok := rs.sessionsTrees[sessionEndHeight]

	// If there is no map for sessions at the sessionEndHeight, create one.
	if !ok {
		sessionsTrees = make(map[sessionId]SessionTree)
		rs.sessionsTrees[sessionEndHeight] = sessionsTrees
	}

	// Get the sessionTree for the given session.
	sessionTree, ok := sessionsTrees[session.SessionId]

	// If the sessionTree does not exist, create it.
	if !ok {
		sessionTree, err := NewSessionTree(session, rs.storesDirectory, rs.removeFromRelayerSessions)
		if err != nil {
			return nil, err
		}

		sessionsTrees[session.SessionId] = sessionTree
	}

	return sessionTree, nil
}

// goListenToCommittedBlocks listens to committed blocks so that rs.closingSessionsPublisher can notify
// when sessions are ready to be claimed.
// It is a background goroutine.
func (rs *relayerSessionsManager) goListenToCommittedBlocks(ctx context.Context) {
	committedBlocks := rs.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx).Ch()

	for block := range committedBlocks {
		// Check if there are sessions to be closed at this block height.
		if sessionsTrees, ok := rs.sessionsTrees[block.Height()]; ok {
			// Iterate over the sessionsTrees that end at this block height and publish them.
			for _, sessionTree := range sessionsTrees {
				rs.closingSessionsPublisher <- sessionTree
			}
		}
	}
}

// removeFromRelayerSessions removes the session from the relayerSessions.
func (rs *relayerSessionsManager) removeFromRelayerSessions(session *sessiontypes.Session) {
	rs.sessionsTreesMu.Lock()
	defer rs.sessionsTreesMu.Unlock()

	sessionEndHeight := session.Header.SessionStartBlockHeight + session.NumBlocksPerSession
	sessionsTrees, ok := rs.sessionsTrees[sessionEndHeight]
	if !ok {
		log.Print("session not found in relayerSessionsManager")
		return
	}

	delete(sessionsTrees, session.SessionId)
}
