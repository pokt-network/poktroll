package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pocket/x/session/types"
)

func (k Keeper) GetSession(goCtx context.Context, req *types.QueryGetSessionRequest) (*types.QueryGetSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	sessionHydrator := NewSessionHydrator(req.ApplicationAddress, req.ServiceId, req.BlockHeight)

	session, err := k.hydrateSession(ctx, sessionHydrator)
	if err != nil {
		return nil, err
	}

	fmt.Println(session)
	return nil, nil
}
