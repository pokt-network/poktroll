package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UpdateParam(ctx context.Context, msg *apptypes.MsgUpdateParam) (*apptypes.MsgUpdateParamResponse, error) {
	logger := k.logger.With(
		"method", "UpdateParam",
		"param_name", msg.Name,
	)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.InvalidArgument,
			apptypes.ErrAppInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case apptypes.ParamMaxDelegatedGateways:
		logger = logger.With("param_value", msg.GetAsUint64())

		params.MaxDelegatedGateways = msg.GetAsUint64()
		if _, ok := msg.AsType.(*apptypes.MsgUpdateParam_AsUint64); !ok {
			return nil, apptypes.ErrAppParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
		}
		maxDelegatedGateways := msg.GetAsUint64()

		if err := apptypes.ValidateMaxDelegatedGateways(maxDelegatedGateways); err != nil {
			return nil, apptypes.ErrAppParamInvalid.Wrapf("maxdelegegated_gateways (%d): %v", maxDelegatedGateways, err)
		}
		params.MaxDelegatedGateways = maxDelegatedGateways
	case apptypes.ParamMinStake:
		logger = logger.With("param_value", msg.GetAsCoin())
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			apptypes.ErrAppParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
	if err := params.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())

	}

	if err := k.SetParams(ctx, params); err != nil {
		err = fmt.Errorf("unable to set params: %w", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	updatedParams := k.GetParams(ctx)

	return &apptypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
