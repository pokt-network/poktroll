package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SharedQueryClient = (*SharedKeeperQueryClient)(nil)

// SharedKeeperQueryClient is a thin wrapper around the SharedKeeper.
// It does not rely on the QueryClient, and therefore does not make any
// network requests as in the offchain implementation.
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

// GetParams queries & returns the shared module onchain parameters.
func (sqc *SharedKeeperQueryClient) GetParams(
	ctx context.Context,
) (params *sharedtypes.Params, err error) {
	sharedParams := sqc.sharedKeeper.GetParams(ctx)
	return &sharedParams, nil
}

// GetParamsAtHeight returns the shared params that were effective at queryHeight.
func (sqc *SharedKeeperQueryClient) GetParamsAtHeight(
	ctx context.Context,
	queryHeight int64,
) (*sharedtypes.Params, error) {
	sharedParams := sqc.sharedKeeper.GetParamsAtHeight(ctx, queryHeight)
	return &sharedParams, nil
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session which includes queryHeight elapses.
// The grace period is the number of blocks after the session ends during which relays
// SHOULD be included in the session which most recently ended.
//
// TODO_MAINNET_MIGRATION(@red-0ne, #543): We don't really want to use the current value of the params.
// Instead, we should be using the value that the params had for the session given by blockHeight.
func (sqc *SharedKeeperQueryClient) GetSessionGracePeriodEndHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParamsAtHeight(ctx, queryHeight)
	return sharedtypes.GetSessionGracePeriodEndHeight(&sharedParams, queryHeight), nil
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens.
func (sqc *SharedKeeperQueryClient) GetClaimWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParamsAtHeight(ctx, queryHeight)
	return sharedtypes.GetClaimWindowOpenHeight(&sharedParams, queryHeight), nil
}

// GetProofWindowOpenHeight returns the block height at which the proof window of
// the session that includes queryHeight opens.
func (sqc *SharedKeeperQueryClient) GetProofWindowOpenHeight(
	ctx context.Context,
	queryHeight int64,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParamsAtHeight(ctx, queryHeight)
	return sharedtypes.GetProofWindowOpenHeight(&sharedParams, queryHeight), nil
}

// GetEarliestSupplierClaimCommitHeight returns the earliest block height at which a claim
// for the session that includes queryHeight can be committed for a given supplier.
func (sqc *SharedKeeperQueryClient) GetEarliestSupplierClaimCommitHeight(
	ctx context.Context,
	queryHeight int64,
	supplierOperatorAddr string,
) (int64, error) {
	sharedParams := sqc.sharedKeeper.GetParamsAtHeight(ctx, queryHeight)

	// CONSENSUS HARDENING — do NOT read the claim-window-open block hash here.
	// The claimWindowOpenBlockHash arg to sharedtypes.GetEarliestSupplierClaimCommitHeight
	// is unused (claim distribution seeding is disabled), so the value is discarded — yet
	// GetBlockHash is a gas-metered store read on the MsgCreateClaim (FinalizeBlock) path.
	// A discarded read that still consumes consensus gas is a latent nondeterminism
	// surface: if it ever returned a different byte length across nodes, gas_used would
	// diverge and LastResultsHash split while AppHash stayed identical. That is the
	// signature of the beta-lego block-432943 halt (transient, self-healed on re-exec);
	// this read is a suspected carrier, not a proven root cause. Passing nil removes the
	// surface. Re-add only if distribution seeding is re-enabled deterministically on AND
	// off chain.
	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		queryHeight,
		nil,
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
	sharedParams := sqc.sharedKeeper.GetParamsAtHeight(ctx, queryHeight)

	// CONSENSUS HARDENING — do NOT read the proof-window-open block hash here; see the
	// detailed note in GetEarliestSupplierClaimCommitHeight above. The block-hash arg is
	// unused (proof distribution seeding disabled), so a gas-metered GetBlockHash read on
	// this on-chain path adds consensus gas for a discarded value — a latent
	// nondeterminism surface with no upside. Pass nil.
	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		queryHeight,
		nil,
		supplierOperatorAddr,
	), nil
}
