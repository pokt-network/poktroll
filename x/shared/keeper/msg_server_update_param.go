package keeper

import (
	"context"
	"fmt"

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
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrSharedParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	msgUpdateParams := &types.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
