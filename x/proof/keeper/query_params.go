package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ParamsAtHeight queries the parameters that were active at a specific block height.
func (k Keeper) ParamsAtHeight(goCtx context.Context, req *types.QueryParamsAtHeightRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get all parameter updates and find the one active at the requested height
	paramsUpdates := k.GetParamsUpdates(ctx)
	activeParamsUpdate := sharedtypes.GetActiveParamsUpdate(paramsUpdates, req.AtHeight)

	return &types.QueryParamsResponse{
		Params:             activeParamsUpdate.Params,
		ActivationHeight:   activeParamsUpdate.ActivationHeight,
		DeactivationHeight: activeParamsUpdate.DeactivationHeight,
	}, nil
}

// Params queries the current params.
// * Returns the parameters that are effective at the current block height
func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := ctx.BlockHeight()

	// Get all parameter updates and find the one active at the current height
	paramsUpdates := k.GetParamsUpdates(ctx)
	activeParamsUpdate := sharedtypes.GetActiveParamsUpdate(paramsUpdates, currentHeight)

	return &types.QueryParamsResponse{
		Params:             activeParamsUpdate.Params,
		ActivationHeight:   activeParamsUpdate.ActivationHeight,
		DeactivationHeight: activeParamsUpdate.DeactivationHeight,
	}, nil
}

// ParamsUpdates queries all parameter updates that have been made in the proof module.
func (k Keeper) ParamsUpdates(
	goCtx context.Context,
	req *types.QueryParamsUpdatesRequest,
) (*types.QueryParamsUpdatesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	paramsUpdates := k.GetParamsUpdates(ctx)

	return &types.QueryParamsUpdatesResponse{
		ParamsUpdates: paramsUpdates,
	}, nil
}
