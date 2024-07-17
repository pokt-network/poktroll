package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
)

func (k Keeper) Params(
	ctx context.Context,
	req *tokenomics.QueryParamsRequest,
) (*tokenomics.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &tokenomics.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
