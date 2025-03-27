package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// UpdateParam updates a single parameter in the shared module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *sharedtypes.MsgUpdateParam,
) (*sharedtypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case sharedtypes.ParamNumBlocksPerSession:
		logger = logger.With("num_blocks_per_session", msg.GetAsUint64())
		params.NumBlocksPerSession = msg.GetAsUint64()
	case sharedtypes.ParamGracePeriodEndOffsetBlocks:
		logger = logger.With("grace_period_end_offset_blocks", msg.GetAsUint64())
		params.GracePeriodEndOffsetBlocks = msg.GetAsUint64()
	case sharedtypes.ParamClaimWindowOpenOffsetBlocks:
		logger = logger.With("claim_window_open_offset_blocks", msg.GetAsUint64())
		params.ClaimWindowOpenOffsetBlocks = msg.GetAsUint64()
	case sharedtypes.ParamClaimWindowCloseOffsetBlocks:
		logger = logger.With("claim_window_close_offset_blocks", msg.GetAsUint64())
		params.ClaimWindowCloseOffsetBlocks = msg.GetAsUint64()
	case sharedtypes.ParamProofWindowOpenOffsetBlocks:
		logger = logger.With("proof_window_open_offset_blocks", msg.GetAsUint64())
		params.ProofWindowOpenOffsetBlocks = msg.GetAsUint64()
	case sharedtypes.ParamProofWindowCloseOffsetBlocks:
		logger = logger.With("proof_window_close_offset_blocks", msg.GetAsUint64())
		params.ProofWindowCloseOffsetBlocks = msg.GetAsUint64()
	case sharedtypes.ParamSupplierUnbondingPeriodSessions:
		logger = logger.With("supplier_unbonding_period_sessions", msg.GetAsUint64())
		params.SupplierUnbondingPeriodSessions = msg.GetAsUint64()
	case sharedtypes.ParamApplicationUnbondingPeriodSessions:
		logger = logger.With("application_unbonding_period_sessions", msg.GetAsUint64())
		params.ApplicationUnbondingPeriodSessions = msg.GetAsUint64()
	case sharedtypes.ParamGatewayUnbondingPeriodSessions:
		logger = logger.With("gateway_unbonding_period_sessions", msg.GetAsUint64())
		params.GatewayUnbondingPeriodSessions = msg.GetAsUint64()
	case sharedtypes.ParamComputeUnitsToTokensMultiplier:
		logger = logger.With("compute_units_to_tokens_multiplier", msg.GetAsUint64())
		params.ComputeUnitsToTokensMultiplier = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			sharedtypes.ErrSharedParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &sharedtypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &sharedtypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
