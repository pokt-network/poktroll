package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/x/shared"
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

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_BLOCKER(#543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sharedQueryClient *SharedKeeperQueryClient) GetClaimWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sharedQueryClient.keeper.GetParams(ctx)
	return shared.GetClaimWindowOpenHeight(&sharedParams, queryHeight), nil
}
