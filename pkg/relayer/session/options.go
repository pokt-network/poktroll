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

// WithSessionTreesInspector exposes the session trees map so that it can be inspected.
// This is useful for testing purposes, as it allows inspecting the session trees present in memory.
func WithSessionTreesInspector(sessionTreeMap *SessionsTreesMap) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		*sessionTreeMap = relSessionMgr.(*relayerSessionsManager).sessionsTrees
	}
}
