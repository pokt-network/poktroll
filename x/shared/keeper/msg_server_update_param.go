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

	// Record the new params in history with their effective height.
	// New params become effective at the start of the next session.
	if err := k.recordParamsHistory(ctx, params); err != nil {
		err = fmt.Errorf("unable to record session params history: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := k.SetParams(ctx, params); err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamResponse{}, nil
}

// recordParamsHistory ensures session params history is properly tracked.
// It initializes history with genesis params if needed, then records new params
// with their effective height (next session start).
func (k msgServer) recordParamsHistory(ctx context.Context, newParams types.Params) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Get the OLD params before we update (these are the currently effective params)
	oldParams := k.GetParams(ctx)

	// Check if history is empty (first param update since genesis or upgrade)
	history := k.GetAllParamsHistory(ctx)
	if len(history) == 0 {
		// Initialize history with the current (old) params at height 1.
		// These params have been effective since genesis.
		if err := k.SetParamsAtHeight(ctx, 1, oldParams); err != nil {
			return fmt.Errorf("failed to initialize session params history: %w", err)
		}
	}

	// Calculate when the new params become effective: start of next session.
	// Use the OLD params for this calculation since they're still in effect.
	currentSessionEndHeight := types.GetSessionEndHeight(&oldParams, currentHeight)
	nextSessionStartHeight := currentSessionEndHeight + 1

	// Store the new params with their effective height.
	if err := k.SetParamsAtHeight(ctx, nextSessionStartHeight, newParams); err != nil {
		return fmt.Errorf("failed to record new session params: %w", err)
	}

	return nil
}
