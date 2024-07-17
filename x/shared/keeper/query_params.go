package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/shared"
)

func (k Keeper) Params(goCtx context.Context, req *shared.QueryParamsRequest) (*shared.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &shared.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
