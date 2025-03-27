package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// UpdateParam updates a single parameter in the supplier module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *suppliertypes.MsgUpdateParam,
) (*suppliertypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case suppliertypes.ParamMinStake:
		logger = logger.With("min_stake", msg.GetAsCoin())
		params.MinStake = msg.GetAsCoin()
	case suppliertypes.ParamStakingFee:
		logger = logger.With("staking_fee", msg.GetAsCoin())
		params.StakingFee = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			suppliertypes.ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &suppliertypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &suppliertypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
