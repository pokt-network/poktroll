package session

import (
	"net/url"

	"github.com/pokt-network/poktroll/pkg/relayer"
)

// WithStoresDirectory sets the path on disk where KVStore data files used to store
// SMST of work sessions are created.
func WithStoresDirectory(storesDirectory string) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		relSessionMgr.(*relayerSessionsManager).storesDirectory = storesDirectory
	}
}

// TODO_IN_THIS_COMMIT: godoc comment.
func WithQueryNodeGRPCUrl(queryNodeGRPCUrl *url.URL) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		relSessionMgr.(*relayerSessionsManager).queryNodeGRPCUrl = queryNodeGRPCUrl
	}
}
