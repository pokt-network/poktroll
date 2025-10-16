package session

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

// WithStoresDirectoryPath sets the path on disk where KVStore data files used to store
// SMST of work sessions are created.
func WithStoresDirectoryPath(storesDirectoryPath string) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		relSessionMgr.(*relayerSessionsManager).storesDirectoryPath = storesDirectoryPath
	}
}
