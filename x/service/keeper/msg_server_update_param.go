package keeper

import (
	"context"

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

	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.InvalidArgument,
			servicetypes.ErrServiceInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case servicetypes.ParamAddServiceFee:
		logger = logger.With("param_value", msg.GetAsCoin())
		params.AddServiceFee = msg.GetAsCoin()
	case servicetypes.ParamTargetNumRelays:
		logger = logger.With("param_value", msg.GetAsUint64())
		params.TargetNumRelays = msg.GetAsUint64()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			servicetypes.ErrServiceParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
	if err := params.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := k.SetParams(ctx, params); err != nil {
		logger.Info("ERROR: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	updatedParams := k.GetParams(ctx)

	return &servicetypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
