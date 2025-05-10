package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/shared/types"
)

// ParamsAtHeight queries the params at a specific height.
func (k Keeper) ParamsAtHeight(goCtx context.Context, req *types.QueryParamsAtHeightRequest) (*types.QueryParamsAtHeightResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	paramsUpdates := k.GetParamsUpdates(ctx)
	paramsAtHeight := types.GetEffectiveParamsUpdate(paramsUpdates, int64(req.AtHeight))

	return &types.QueryParamsAtHeightResponse{
		Params:               paramsAtHeight.Params,
		EffectiveBlockHeight: paramsAtHeight.EffectiveBlockHeight,
	}, nil
}

// Params queries the current params.
// This is the params that are effective at the current block height.
func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	lastCommitHeight := ctx.BlockHeight()

	paramsUpdates := k.GetParamsUpdates(ctx)
	paramsAtHeight := types.GetEffectiveParamsUpdate(paramsUpdates, lastCommitHeight)

	return &types.QueryParamsResponse{
		Params:               paramsAtHeight.Params,
		EffectiveBlockHeight: paramsAtHeight.EffectiveBlockHeight,
	}, nil
}

func (k Keeper) ParamsUpdates(goCtx context.Context, req *types.QueryParamsUpdatesRequest) (*types.QueryParamsUpdatesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	paramsUpdates := k.GetParamsUpdates(ctx)

	return &types.QueryParamsUpdatesResponse{
		ParamsUpdates: paramsUpdates,
	}, nil
}
