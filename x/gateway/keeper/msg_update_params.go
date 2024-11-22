package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) UpdateParams(
	goCtx context.Context,
	req *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if k.GetAuthority() != req.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrGatewayInvalidSigner.Wrapf(
				"invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority,
			).Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	// NOTE(#322): Omitted parameters will be set to their zero value.
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
