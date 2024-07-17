package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/proof"
)

func (k Keeper) Params(ctx context.Context, req *proof.QueryParamsRequest) (*proof.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &proof.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
