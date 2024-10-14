package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(ctx context.Context, msg *suppliertypes.MsgUpdateParam) (*suppliertypes.MsgUpdateParamResponse, error) {
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
			suppliertypes.ErrSupplierInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.Authority,
			).Error(),
		)
	}

	params := k.GetParams(ctx)

	switch msg.Name {
	case suppliertypes.ParamMinStake:
		logger = logger.With("min_stake", msg.GetAsCoin())
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			suppliertypes.ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	// Perform a global validation on all params, which includes the updated param.
	// This is needed to ensure that the updated param is valid in the context of all other params.
	if err := params.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := k.SetParams(ctx, params); err != nil {
		err = fmt.Errorf("unable to set params: %v", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	updatedParams := k.GetParams(ctx)

	return &suppliertypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
