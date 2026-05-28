package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/shared/types"
)

func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// ParamsAtHeight returns the shared params that were effective at the requested height.
// Off-chain clients use this to compute a session's claim/proof windows with the
// num_blocks_per_session that was in effect when that session started (#543 anchored grid).
func (k Keeper) ParamsAtHeight(goCtx context.Context, req *types.QueryParamsAtHeightRequest) (*types.QueryParamsAtHeightResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Height < 0 {
		return nil, status.Error(codes.InvalidArgument, "height must be non-negative")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryParamsAtHeightResponse{Params: k.GetParamsAtHeight(ctx, req.Height)}, nil
}
