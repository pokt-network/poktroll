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
	sharedKeeper  SharedKeeper
	sessionKeeper SessionKeeper
}

// NewSharedKeeperQueryClient returns a new SharedQueryClient that is backed
// by an SharedKeeper instance.
func NewSharedKeeperQueryClient(
	sharedKeeper SharedKeeper,
	sessionKeeper SessionKeeper,
) client.SharedQueryClient {
	return &SharedKeeperQueryClient{
		sharedKeeper:  sharedKeeper,
		sessionKeeper: sessionKeeper,
	}
}

// GetParams queries & returns the shared module on-chain parameters.
func (sqc *SharedKeeperQueryClient) GetParams(
	ctx context.Context,
) (params *sharedtypes.Params, err error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	return &sharedParams, nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
//
// TODO_BLOCKER(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	return shared.GetSessionGracePeriodEndHeight(&sharedParams, queryHeight), nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_BLOCKER(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetClaimWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	return shared.GetClaimWindowOpenHeight(&sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
//
// TODO_BLOCKER(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetProofWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	return shared.GetProofWindowOpenHeight(&sharedParams, queryHeight), nil
}

func (sqc *SharedKeeperQueryClient) GetEarliestClaimCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierAddr string,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	claimWindowOpenHeight := shared.GetClaimWindowOpenHeight(&sharedParams, queryHeight)

	// Fetch the claim window open block hash so that it can be used as part of the
	// pseudo-random seed for generating the claim distribution offset.
	// NB: Raw byte slice representations of block hashes don't need to be normalized.
	claimWindowOpenBlockHashBz := sqc.sessionKeeper.GetBlockHash(ctx, claimWindowOpenHeight)

	// Get the earliest claim commit height for the given supplier.
	return shared.GetEarliestClaimCommitHeight(
		&sharedParams,
		queryHeight,
		claimWindowOpenBlockHashBz,
		supplierAddr,
	), nil
}

func (sqc *SharedKeeperQueryClient) GetEarliestProofCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierAddr string,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(&sharedParams, queryHeight)

	// Fetch the proof window open block hash so that it can be used as part of the
	// pseudo-random seed for generating the proof distribution offset.
	// NB: Raw byte slice representations of block hashes don't need to be normalized.
	proofWindowOpenBlockHash := sqc.sessionKeeper.GetBlockHash(ctx, proofWindowOpenHeight)

	// Get the earliest proof commit height for the given supplier.
	return shared.GetEarliestProofCommitHeight(
		&sharedParams,
		queryHeight,
		proofWindowOpenBlockHash,
		supplierAddr,
	), nil
}
