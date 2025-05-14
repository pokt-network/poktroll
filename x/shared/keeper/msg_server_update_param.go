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
// * Validates the request message and authority permissions
// * Updates the specific parameter based on its name
// * Delegates to UpdateParams to handle validation and persistence
// * Returns both the current parameters and the scheduled parameter update
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *sharedtypes.MsgUpdateParam,
) (*sharedtypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	// Validate basic message structure and constraints
	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get current parameters to apply the single parameter update
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

	// Create a full params update message and delegate to UpdateParams
	// This ensures:
	// * Authority validation
	// * Parameter constraints validation
	msgUpdateParams := &sharedtypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, err
	}

	// Return a response with both the current parameters and the scheduled update
	// This allows the caller to see the current state and the scheduled change
	return &sharedtypes.MsgUpdateParamResponse{
		Params:       response.Params,
		ParamsUpdate: response.ParamsUpdate,
	}, nil
}
