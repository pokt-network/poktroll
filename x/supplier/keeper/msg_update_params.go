package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) UpdateParams(
	ctx context.Context,
	req *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger().With("method", "UpdateParams")

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if k.GetAuthority() != req.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrSupplierInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority,
			).Error(),
		)
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		err = status.Error(codes.Internal, err.Error())
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
