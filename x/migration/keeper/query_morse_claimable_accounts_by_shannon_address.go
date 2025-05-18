package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// TODO_IN_THIS_PR(@bryanchriswhite): Comment the use case
func (k Keeper) MorseClaimableAccountsByShannonAddress(goCtx context.Context, req *types.QueryMorseClaimableAccountsByShannonAddressRequest) (*types.QueryMorseClaimableAccountsByShannonAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO_IN_THIS_PR(@bryanchriswhite): Implement this query
	_ = ctx

	return &types.QueryMorseClaimableAccountsByShannonAddressResponse{}, nil
}
