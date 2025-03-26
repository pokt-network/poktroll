package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// UpdateParam updates a single parameter in the application module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *apptypes.MsgUpdateParam,
) (*apptypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case apptypes.ParamMaxDelegatedGateways:
		logger = logger.With("max_delegated_gateways", msg.GetAsUint64())
		params.MaxDelegatedGateways = msg.GetAsUint64()
	case apptypes.ParamMinStake:
		logger = logger.With("min_stake", msg.GetAsCoin())
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			apptypes.ErrAppParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &apptypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, err
	}

	return &apptypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
