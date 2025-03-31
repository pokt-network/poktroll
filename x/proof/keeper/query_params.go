package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// ParamsAtHeight queries the params at a specific height.
func (k Keeper) ParamsAtHeight(goCtx context.Context, req *types.QueryParamsAtHeightRequest) (*types.QueryParamsAtHeightResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	paramsUpdates := k.GetParamsUpdates(ctx)
	paramsAtHeight := getEffectiveParamsUpdate(paramsUpdates, int64(req.Height))

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
	paramsAtHeight := getEffectiveParamsUpdate(paramsUpdates, int64(lastCommitHeight))

	return &types.QueryParamsResponse{
		Params:               paramsAtHeight.Params,
		EffectiveBlockHeight: paramsAtHeight.EffectiveBlockHeight,
	}, nil
}

// getEffectiveParamsUpdate returns the effective params update as of the query height.
func getEffectiveParamsUpdate(sharedParamsUpdates []types.ParamsUpdate, queryHeight int64) types.ParamsUpdate {
	var effectiveParamsUpdate types.ParamsUpdate
	for _, update := range sharedParamsUpdates {
		// The params updates are chronologically ordered from the oldest to the most recent.
		// We can stop iterating when we find the first params update that is effective
		// after the query height.
		if update.EffectiveBlockHeight > uint64(queryHeight) {
			break
		}

		effectiveParamsUpdate = update
	}

	return effectiveParamsUpdate
}
