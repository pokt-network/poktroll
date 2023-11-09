package session

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

// WithStoresDirectory sets the path on disk where KVStore data files are created.
func WithStoresDirectory(storesDirectory string) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		relSessionMgr.(*relayerSessionsManager).storesDirectory = storesDirectory
	}
}
