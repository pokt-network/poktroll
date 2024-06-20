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

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_BLOCKER(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sharedQueryClient *SharedKeeperQueryClient) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sharedQueryClient.keeper.GetParams(ctx)
	return shared.GetSessionGracePeriodEndHeight(&sharedParams, queryHeight), nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_BLOCKER(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sharedQueryClient *SharedKeeperQueryClient) GetClaimWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sharedQueryClient.keeper.GetParams(ctx)
	return shared.GetClaimWindowOpenHeight(&sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
//
// TODO_BLOCKER(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sharedQueryClient *SharedKeeperQueryClient) GetProofWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sharedQueryClient.keeper.GetParams(ctx)
	return shared.GetProofWindowOpenHeight(&sharedParams, queryHeight), nil
}
