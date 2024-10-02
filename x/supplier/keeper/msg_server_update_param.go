package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// UpdateParam updates a single parameter in the proof module and returns
// all active parameters.
func (k msgServer) UpdateParam(ctx context.Context, msg *suppliertypes.MsgUpdateParam) (*suppliertypes.MsgUpdateParamResponse, error) {
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
		params.MinStake = msg.GetAsCoin()
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			suppliertypes.ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
		)
	}

	if err := k.SetParams(ctx, params); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	updatedParams := k.GetParams(ctx)
	return &suppliertypes.MsgUpdateParamResponse{
		Params: &updatedParams,
	}, nil
}
