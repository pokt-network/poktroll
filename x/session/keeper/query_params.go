package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/session"
)

func (k Keeper) Params(
	ctx context.Context,
	req *session.QueryParamsRequest,
) (*session.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	return &session.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
