package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pocket/x/session/types"
)

func (k Keeper) GetSession(goCtx context.Context, req *types.QueryGetSessionRequest) (*types.QueryGetSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// If block height is not specified, use the current (context's latest) block height
	blockHeight := req.BlockHeight
	if blockHeight == 0 {
		blockHeight = ctx.BlockHeight()
	}

	sessionHydrator := NewSessionHydrator(req.ApplicationAddress, req.ServiceId.Id, blockHeight)
	session, err := k.HydrateSession(ctx, sessionHydrator)
	if err != nil {
		return nil, err
	}

	res := &types.QueryGetSessionResponse{
		Session: session,
	}
	return res, nil
}
