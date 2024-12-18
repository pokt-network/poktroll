package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SharedQueryClient = (*sharedQuerier)(nil)

// sharedQuerier is a wrapper around the sharedtypes.QueryClient that enables the
// querying of on-chain shared information.
type sharedQuerier struct {
	client.ParamsQuerier[*sharedtypes.Params]

	clientConn    grpc.ClientConn
	sharedQuerier sharedtypes.QueryClient
	blockQuerier  client.BlockQueryClient
}

// NewSharedQuerier returns a new instance of a client.SharedQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - client.BlockQueryClient
func NewSharedQuerier(
	deps depinject.Config,
	paramsQuerierOpts ...ParamsQuerierOptionFn,
) (client.SharedQueryClient, error) {
	paramsQuerierCfg := DefaultParamsQuerierConfig()
	for _, opt := range paramsQuerierOpts {
		opt(paramsQuerierCfg)
	}

	paramsQuerier, err := NewCachedParamsQuerier[*sharedtypes.Params, sharedtypes.SharedQueryClient](
		deps, sharedtypes.NewSharedQueryClient,
		WithModuleInfo(sharedtypes.ModuleName, sharedtypes.ErrSharedParamInvalid),
		WithQueryCacheOptions(paramsQuerierCfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	sq := &sharedQuerier{
		ParamsQuerier: paramsQuerier,
	}

	if err = depinject.Inject(
		deps,
		&sq.clientConn,
		&sq.blockQuerier,
	); err != nil {
		return nil, err
	}

	sq.sharedQuerier = sharedtypes.NewQueryClient(sq.clientConn)

	return sq, nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
func (sq *sharedQuerier) GetClaimWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
func (sq *sharedQuerier) GetProofWindowOpenHeight(ctx context.Context, queryHeight int64) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedtypes.GetSessionGracePeriodEndHeight(sharedParams, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierClaimCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierOperatorAddr string,
) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight)
	claimWindowOpenBlock, err := sq.blockQuerier.Block(ctx, &claimWindowOpenHeight)
	if err != nil {
		return 0, err
	}

	// NB: Byte slice representation of block hashes don't need to be normalized.
	claimWindowOpenBlockHash := claimWindowOpenBlock.BlockID.Hash.Bytes()

	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		queryHeight,
		claimWindowOpenBlockHash,
		supplierOperatorAddr,
	), nil
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier.
//
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetEarliestSupplierProofCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierOperatorAddr string,
) (int64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	// Fetch the block at the proof window open height. Its hash is used as part
	// of the seed to the pseudo-random number generator.
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight)
	proofWindowOpenBlock, err := sq.blockQuerier.Block(ctx, &proofWindowOpenHeight)
	if err != nil {
		return 0, err
	}

	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		queryHeight,
		proofWindowOpenBlock.BlockID.Hash,
		supplierOperatorAddr,
	), nil
}

// GetComputeUnitsToTokensMultiplier returns the multiplier used to convert compute units to tokens.
//
// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
// to get the most recently (asynchronously) observed (and cached) value.
// TODO_MAINNET(@bryanchriswhite, #543): We also don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session which includes queryHeight.
func (sq *sharedQuerier) GetComputeUnitsToTokensMultiplier(ctx context.Context, queryHeight int64) (uint64, error) {
	sharedParams, err := sq.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}
	return sharedParams.GetComputeUnitsToTokensMultiplier(), nil
}
