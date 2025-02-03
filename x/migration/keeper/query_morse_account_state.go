package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// MorseAccountState returns the morseAccountState, if one has been created;
// otherwise, an error is returned.
func (k Keeper) MorseAccountState(goCtx context.Context, req *types.QueryGetMorseAccountStateRequest) (*types.QueryGetMorseAccountStateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	morseAccountState, isFound := k.GetMorseAccountState(ctx)
	if !isFound {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetMorseAccountStateResponse{MorseAccountState: morseAccountState}, nil
}
