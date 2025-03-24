package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/pocket/x/shared/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger().With("method", "UpdateParams")

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if k.GetAuthority() != req.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrSharedInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), req.Authority,
			).Error(),
		)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		err = types.ErrSharedParamInvalid.Wrapf("unable to set params: %v", err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
