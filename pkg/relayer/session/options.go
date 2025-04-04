package session

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

// WithStoresDirectory sets the path on disk where KVStore data files used to store
// SMST of work sessions are created.
func WithStoresDirectory(storesDirectory string) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		relSessionMgr.(*relayerSessionsManager).storesDirectory = storesDirectory
	}
}

// WithSessionTreesInspector allows setting the relay session manager's session tree map via a pointer.
// In other words, it exposes the session trees map for testing purposes.
// This shouldn't be used in production, but useful for testing so internal structures
// can be accessed and validated for expected state.
func WithSessionTreesInspector(sessionTreeMap *SessionsTreesMap) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		*sessionTreeMap = relSessionMgr.(*relayerSessionsManager).sessionsTrees
	}
}
