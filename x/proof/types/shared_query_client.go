package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SharedQueryClient = (*SharedKeeperQueryClient)(nil)

// SharedKeeperQueryClient is a thin wrapper around the SharedKeeper.
// It does not rely on the QueryClient, and therefore does not make any
// network requests as in the off-chain implementation.
type SharedKeeperQueryClient struct {
	*keeper.KeeperParamsQuerier[sharedtypes.Params, SharedKeeper]

	sharedKeeper  SharedKeeper
	sessionKeeper SessionKeeper
}

// NewSharedKeeperQueryClient returns a new SharedQueryClient that is backed
// by an SharedKeeper instance.
func NewSharedKeeperQueryClient(
	sharedKeeper SharedKeeper,
	sessionKeeper SessionKeeper,
) client.SharedQueryClient {
	keeperParamsQuerier := keeper.NewKeeperParamsQuerier[sharedtypes.Params](sharedKeeper)

	return &SharedKeeperQueryClient{
		KeeperParamsQuerier: keeperParamsQuerier,
		sharedKeeper:        sharedKeeper,
		sessionKeeper:       sessionKeeper,
	}
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_MAINNET(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sqc.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	return sharedtypes.GetSessionGracePeriodEndHeight(sharedParams, queryHeight), nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
//
// TODO_MAINNET(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetClaimWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sqc.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	return sharedtypes.GetClaimWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
//
// TODO_MAINNET(@bryanchriswhite, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetProofWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams, err := sqc.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	return sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
func (sqc *SharedKeeperQueryClient) GetEarliestSupplierClaimCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierOperatorAddr string,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&sharedParams, queryHeight)

	// Fetch the claim window open block hash so that it can be used as part of the
	// pseudo-random seed for generating the claim distribution offset.
	// NB: Raw byte slice representations of block hashes don't need to be normalized.
	claimWindowOpenBlockHashBz := sqc.sessionKeeper.GetBlockHash(ctx, claimWindowOpenHeight)

	// Get the earliest claim commit height for the given supplier.
	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		queryHeight,
		claimWindowOpenBlockHashBz,
		supplierOperatorAddr,
	), nil
}

// GetEarliestSupplierProofCommitHeight returns the earliest block height at which a proof
// for the session that includes queryHeight can be committed for a given supplier.
func (sqc *SharedKeeperQueryClient) GetEarliestSupplierProofCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierOperatorAddr string,
) (int64, error) {
	sharedParams, err := sqc.GetParamsAtHeight(ctx, queryHeight)
	if err != nil {
		return 0, err
	}

	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, queryHeight)

	// Fetch the proof window open block hash so that it can be used as part of the
	// pseudo-random seed for generating the proof distribution offset.
	// NB: Raw byte slice representations of block hashes don't need to be normalized.
	proofWindowOpenBlockHash := sqc.sessionKeeper.GetBlockHash(ctx, proofWindowOpenHeight)

	// Get the earliest proof commit height for the given supplier.
	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		queryHeight,
		proofWindowOpenBlockHash,
		supplierOperatorAddr,
	), nil
}

// GetComputeUnitsToTokensMultiplier returns the multiplier used to convert compute units to tokens.
//
// TODO_POST_MAINNNET: If this changes mid-session, the cost of the relays at the
// end of the session may differ from what was anticipated at the beginning.
// Since this will be a non-frequent occurrence, accounting for this edge case is
// not an immediate blocker.
func (sqc *SharedKeeperQueryClient) GetComputeUnitsToTokensMultiplier(ctx context.Context) (uint64, error) {
	sharedParams, err := sqc.GetParamsAtHeight(ctx, 0)
	if err != nil {
		return 0, err
	}

	return sharedParams.GetComputeUnitsToTokensMultiplier(), nil
}
