package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/supplier"
)

func (k Keeper) Params(
	ctx context.Context,
	req *supplier.QueryParamsRequest,
) (*supplier.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &supplier.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
