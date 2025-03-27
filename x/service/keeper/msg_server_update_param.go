package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// UpdateParam updates a single parameter in the service module and returns
// all active parameters.
func (k msgServer) UpdateParam(
	ctx context.Context,
	msg *servicetypes.MsgUpdateParam,
) (*servicetypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case servicetypes.ParamAddServiceFee:
		logger = logger.With("add_service_fee", msg.GetAsCoin())
		params.AddServiceFee = msg.GetAsCoin()
	case servicetypes.ParamTargetNumRelays:
		logger = logger.With("target_num_relays", msg.GetAsUint64())
		params.TargetNumRelays = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			servicetypes.ErrServiceParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Reconstruct a full params update request and rely on the UpdateParams method
	// to handle the authority and basic validation checks of the params.
	msgUpdateParams := &servicetypes.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}
	response, err := k.UpdateParams(ctx, msgUpdateParams)
	if err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &servicetypes.MsgUpdateParamResponse{
		Params:               response.Params,
		EffectiveBlockHeight: response.EffectiveBlockHeight,
	}, nil
}
