package sdkadapter

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/shannon-sdk/sdk"

	"github.com/pokt-network/poktroll/pkg/client"
)

var _ sdk.BlockClient = (*sdkBlockClient)(nil)

// sdkBlockClient is used used by the ShannonSDK to query the latest block height.
type sdkBlockClient struct {
	client client.BlockClient
}

// NewBlockClient creates a new ShannonSDK compatible block client.
// It is a wrapper around the client.BlockClient and implements the sdk.BlockClient
// interface.
//
// Required dependencies:
// - shannonsdk.BlockClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
) (sdk.BlockClient, error) {
	blockClient := &sdkBlockClient{}

	if err := depinject.Inject(deps, &blockClient.client); err != nil {
		return nil, err
	}

	return blockClient, nil
}

// GetLatestBlockHeight returns the latest committed block height.
func (bc *sdkBlockClient) GetLatestBlockHeight(
	ctx context.Context,
) (height int64, err error) {
	return bc.client.LastBlock(ctx).Height(), nil
}
