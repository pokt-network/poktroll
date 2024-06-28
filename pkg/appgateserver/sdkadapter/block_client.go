package sdkadapter

import (
	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
)

// NewBlockClient creates a new ShannonSDK compatible block client.
func NewBlockClient(deps depinject.Config) (client.BlockClient, error) {
	blockClient := client.BlockClient(nil)

	if err := depinject.Inject(deps, &blockClient); err != nil {
		return nil, err
	}

	return blockClient, nil
}
