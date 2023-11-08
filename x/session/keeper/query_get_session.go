package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/session/types"
)

func (k Keeper) GetSession(goCtx context.Context, req *types.QueryGetSessionRequest) (*types.QueryGetSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	sessionHydrator := NewSessionHydrator(req.ApplicationAddress, req.Service.Id, req.BlockHeight)
	session, err := k.HydrateSession(ctx, sessionHydrator)
	if err != nil {
		return nil, err
	}

	res := &types.QueryGetSessionResponse{
		Session: session,
	}
	return res, nil
}
