package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) UpdateParam(ctx context.Context, msg *types.MsgUpdateParam) (*types.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrSharedInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case types.ParamNumBlocksPerSession:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.NumBlocksPerSession = msg.GetAsUint64()
	case types.ParamGracePeriodEndOffsetBlocks:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.GracePeriodEndOffsetBlocks = msg.GetAsUint64()
	case types.ParamClaimWindowOpenOffsetBlocks:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ClaimWindowOpenOffsetBlocks = msg.GetAsUint64()
	case types.ParamClaimWindowCloseOffsetBlocks:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ClaimWindowCloseOffsetBlocks = msg.GetAsUint64()
	case types.ParamProofWindowOpenOffsetBlocks:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ProofWindowOpenOffsetBlocks = msg.GetAsUint64()
	case types.ParamProofWindowCloseOffsetBlocks:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ProofWindowCloseOffsetBlocks = msg.GetAsUint64()
	case types.ParamSupplierUnbondingPeriodSessions:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.SupplierUnbondingPeriodSessions = msg.GetAsUint64()
	case types.ParamApplicationUnbondingPeriodSessions:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ApplicationUnbondingPeriodSessions = msg.GetAsUint64()
	case types.ParamGatewayUnbondingPeriodSessions:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.GatewayUnbondingPeriodSessions = msg.GetAsUint64()
	case types.ParamComputeUnitsToTokensMultiplier:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ComputeUnitsToTokensMultiplier = msg.GetAsUint64()
	case types.ParamComputeUnitCostGranularity:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.ComputeUnitCostGranularity = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrSharedParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
	if err := params.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Record the new params in history at their effective height (start of the next session)
	// and apply the live write per Option B (#543 anchored grid): a session-timing param change
	// (num_blocks_per_session or any session/claim/proof window offset) is deferred to the
	// EndBlocker so in-flight sessions and claims keep the params they were created under, while
	// any other shared param takes effect on live immediately as before.
	if err := k.recordParamsHistory(ctx, params); err != nil {
		err = fmt.Errorf("unable to record session params history: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamResponse{}, nil
}

// recordParamsHistory ensures session params history is properly tracked, stamps the
// derived anchored-session-grid fields on the new epoch, and records the new params with
// their effective height (start of the next session). It does NOT update live params;
// promotion to live happens in the shared EndBlocker (#543 anchored grid, Option B).
func (k msgServer) recordParamsHistory(ctx context.Context, newParams types.Params) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Seed the genesis epoch in history if empty, anchored at block 1 (the legacy grid).
	// Recording at height 1 (NOT currentHeight) ensures every pre-first-change height
	// resolves to the genesis grid via GetParamsAtHeight, rather than falling back to live
	// params — which, once live carries a future anchor, would hit the §3.4 garbage path.
	if !k.HasParamsHistory(ctx) {
		genesisParams := k.GetParams(ctx)
		genesisParams.SessionGridAnchorHeight = 1
		genesisParams.SessionNumberAtAnchor = 1
		if err := k.SetParamsAtHeight(ctx, 1, genesisParams); err != nil {
			return fmt.Errorf("failed to initialize shared params history: %w", err)
		}
	}

	// Compute the next session boundary from the params effective at the current height
	// (NOT live params: under multi-change-per-session, a prior pending change is already
	// in history and live still describes the current epoch — using at-height is robust to
	// both). New params become effective at the start of the next session.
	effectiveParams := k.GetParamsAtHeight(ctx, currentHeight)
	currentSessionEndHeight := types.GetSessionEndHeight(&effectiveParams, currentHeight)
	nextSessionStartHeight := currentSessionEndHeight + 1

	// Stamp the new epoch's DERIVED grid-anchor metadata (governance-supplied values are
	// overwritten — these fields are not user-settable). The anchor MUST move only when
	// num_blocks_per_session changes: the new N does not align with the old grid, so the
	// anchor records the new starting point. For any OTHER param change, the grid is
	// unchanged and the previous epoch's anchor is mathematically equivalent to a new
	// anchor at nextSessionStartHeight (since nextSessionStartHeight is already on the
	// current grid). Carrying the previous anchor forward keeps the field stable when N
	// is unchanged — surprises observers/tests less and matches the field's "describes
	// the current grid" semantics.
	if newParams.NumBlocksPerSession != effectiveParams.NumBlocksPerSession {
		newParams.SessionGridAnchorHeight = uint64(nextSessionStartHeight)
		newParams.SessionNumberAtAnchor = uint64(types.GetSessionNumber(&effectiveParams, currentHeight) + 1)
	} else {
		newParams.SessionGridAnchorHeight = effectiveParams.SessionGridAnchorHeight
		newParams.SessionNumberAtAnchor = effectiveParams.SessionNumberAtAnchor
	}

	// Always record the new params in history at their effective height (next session
	// boundary). The shared EndBlocker promotes this entry to live when block height reaches
	// nextSessionStartHeight — advancing the grid anchor only when N changed; otherwise the
	// promotion is a no-op for the anchor and updates only the changed param(s).
	if err := k.SetParamsAtHeight(ctx, nextSessionStartHeight, newParams); err != nil {
		return fmt.Errorf("failed to record new session params: %w", err)
	}

	// Option B (#543): defer the LIVE write to the EndBlocker for any SESSION-TIMING param
	// change — num_blocks_per_session and the session/claim/proof window offsets. These params
	// drive session boundary, claim window, and proof window math, all of which settlement
	// reads from LIVE params. If live held the new value before the next boundary, an in-flight
	// session/claim would be re-measured on a different grid or window and misalign — orphaning
	// the claim (num_blocks_per_session: re-gridded session ID; window offsets: a shrunk window
	// can mark a claim expired before its proof was ever submittable). Deferring to the boundary
	// keeps in-flight work on the params it was created under; the EndBlocker promotes the new
	// epoch at nextSessionStartHeight.
	//
	// For every OTHER shared param (unbonding periods, compute-unit economics), preserve the
	// legacy behavior of taking effect on live params immediately, but keep the CURRENT epoch's
	// grid anchor in live (the EndBlocker advances the anchor at the boundary) so boundary math
	// stays on the unchanged grid in the meantime.
	liveParams := k.GetParams(ctx)
	if !sessionTimingParamsChanged(&liveParams, &newParams) {
		immediateParams := newParams
		immediateParams.SessionGridAnchorHeight = liveParams.SessionGridAnchorHeight
		immediateParams.SessionNumberAtAnchor = liveParams.SessionNumberAtAnchor
		if err := k.SetParams(ctx, immediateParams); err != nil {
			return fmt.Errorf("failed to set live params: %w", err)
		}
	}

	return nil
}

// sessionTimingParamsChanged reports whether any session-timing param differs between
// the live and new params. These are the params that feed session boundary, claim
// window, and proof window math; a change to any of them must be deferred to the next
// session boundary (#543 Option B) so in-flight sessions and claims are not re-measured
// mid-flight. Grid-anchor metadata is intentionally excluded (it is derived, not governed).
func sessionTimingParamsChanged(live, next *types.Params) bool {
	return live.NumBlocksPerSession != next.NumBlocksPerSession ||
		live.GracePeriodEndOffsetBlocks != next.GracePeriodEndOffsetBlocks ||
		live.ClaimWindowOpenOffsetBlocks != next.ClaimWindowOpenOffsetBlocks ||
		live.ClaimWindowCloseOffsetBlocks != next.ClaimWindowCloseOffsetBlocks ||
		live.ProofWindowOpenOffsetBlocks != next.ProofWindowOpenOffsetBlocks ||
		live.ProofWindowCloseOffsetBlocks != next.ProofWindowCloseOffsetBlocks
}
