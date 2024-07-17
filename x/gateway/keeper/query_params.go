package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/gateway"
)

func (k Keeper) Params(ctx context.Context, req *gateway.QueryParamsRequest) (*gateway.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &gateway.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
