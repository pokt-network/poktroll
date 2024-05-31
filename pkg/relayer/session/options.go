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

// WithSigningKeyName sets the names of the keys which are then used to get the addresses
// to save them in sessionTree.
func WithSigningKeyNames(keyNames []string) relayer.RelayerSessionsManagerOption {
	return func(sClient relayer.RelayerSessionsManager) {
		sClient.(*relayerSessionsManager).signingKeyNames = keyNames
	}
}
