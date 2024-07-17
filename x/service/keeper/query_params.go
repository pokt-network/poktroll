package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/service"
)

func (k Keeper) Params(ctx context.Context, req *service.QueryParamsRequest) (*service.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &service.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
