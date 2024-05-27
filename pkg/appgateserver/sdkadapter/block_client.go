package sdkadapter

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/shannon-sdk/sdk"

	"github.com/pokt-network/poktroll/pkg/client"
)

var _ sdk.BlockClient = (*sdkBlockClient)(nil)

// sdkBlockClient is wrapper around the client.BlockClient that implements the
// ShannonSDK sdk.BlockClient
type sdkBlockClient struct {
	client client.BlockClient
}

// NewBlockClient creates a new ShannonSDK compatible block client.
// It is a wrapper around the client.BlockClient and implements the sdk.BlockClient
// interface.
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
) (sdk.BlockClient, error) {
	blockClient := &sdkBlockClient{}

	depinject.Inject(deps, &blockClient.client)

	return blockClient, nil
}

// GetLatestBlockHeight returns the latest committed block height.
func (bc *sdkBlockClient) GetLatestBlockHeight(
	ctx context.Context,
) (height int64, err error) {
	return bc.client.LastBlock(ctx).Height(), nil
}
