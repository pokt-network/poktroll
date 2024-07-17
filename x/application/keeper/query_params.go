package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/application"
)

func (k Keeper) Params(
	ctx context.Context,
	req *application.QueryParamsRequest,
) (*application.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &application.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
