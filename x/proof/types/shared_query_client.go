package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SharedQueryClient = (*SharedKeeperQueryClient)(nil)

// SharedKeeperQueryClient is a thin wrapper around the SharedKeeper.
// It does not rely on the QueryClient, and therefore does not make any
// network requests as in the off-chain implementation.
type SharedKeeperQueryClient struct {
	keeper SharedKeeper
}

// NewSharedKeeperQueryClient returns a new SharedQueryClient that is backed
// by an SharedKeeper instance.
func NewSharedKeeperQueryClient(sharedKeeper SharedKeeper) client.SharedQueryClient {
	return &SharedKeeperQueryClient{keeper: sharedKeeper}
}

// GetParams queries & returns the shared module on-chain parameters.
func (sharedQueryClient *SharedKeeperQueryClient) GetParams(
	ctx context.Context,
) (params *sharedtypes.Params, err error) {
	sharedParams := sharedQueryClient.keeper.GetParams(ctx)
	return &sharedParams, nil
}
