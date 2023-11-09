package session

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

func WithStoresDirectory(storesDirectory string) relayer.RelayerSessionsManagerOption {
	return func(relSessionMgr relayer.RelayerSessionsManager) {
		relSessionMgr.(*relayerSessionsManager).storesDirectory = storesDirectory
	}
}
