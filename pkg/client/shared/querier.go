package shared

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/x/shared"
)

var _ client.SharedQueryClient = (*sharedQueryClient)(nil)

// sharedQueryClient is a wrapper around the sharedtypes.QueryClient that enables the
// querying of on-chain shared information through a single exposed method
// which returns an sharedtypes.Session struct
type sharedQueryClient struct {
	clientConn    grpc.ClientConn
	sharedQuerier sharedtypes.QueryClient
	blockQuerier  client.BlockQueryClient
}

// NewSharedQueryClient returns a new instance of a client.SharedQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - client.BlockQueryClient
func NewSharedQueryClient(deps depinject.Config) (client.SharedQueryClient, error) {
	querier := &sharedQueryClient{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.blockQuerier,
	); err != nil {
		return nil, err
	}

	querier.sharedQuerier = sharedtypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetParams queries & returns the shared module on-chain parameters.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
func (sq *sharedQueryClient) GetParams(ctx context.Context) (*sharedtypes.Params, error) {
	req := &sharedtypes.QueryParamsRequest{}
	res, err := sq.sharedQuerier.Params(ctx, req)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}
	return &res.Params, nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_BLOCKER(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
// we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQueryClient) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return shared.GetClaimWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_BLOCKER(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
// we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQueryClient) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return shared.GetProofWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_BLOCKER(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQueryClient) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return shared.GetSessionGracePeriodEndHeight(sharedParams, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_BLOCKER(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQueryClient) GetEarliestSupplierClaimCommitHeight(ctx context.Context, queryHeight int64, supplierAddr string) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	claimWindowOpenHeight := shared.GetClaimWindowOpenHeight(sharedParams, queryHeight)
	claimWindowOpenBlock, err := sq.blockQuerier.Block(ctx, &claimWindowOpenHeight)
	if err != nil {
		return 0, err
	}

	// NB: Byte slice representation of block hashes don't need to be normalized.
	claimWindowOpenBlockHash := claimWindowOpenBlock.BlockID.Hash.Bytes()

	return shared.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		queryHeight,
		claimWindowOpenBlockHash,
		supplierAddr,
	), nil
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_BLOCKER(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQueryClient) GetEarliestSupplierProofCommitHeight(ctx context.Context, queryHeight int64, supplierAddr string) (int64, error) {
	sharedParams, err := sq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(sharedParams, queryHeight)
	proofWindowOpenBlock, err := sq.blockQuerier.Block(ctx, &proofWindowOpenHeight)
	if err != nil {
		return 0, err
	}

	return shared.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		queryHeight,
		proofWindowOpenBlock.BlockID.Hash,
		supplierAddr,
	), nil
}
