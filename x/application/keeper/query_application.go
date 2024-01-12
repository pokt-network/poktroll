package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/application/types"
)

func (k Keeper) ApplicationAll(goCtx context.Context, req *types.QueryAllApplicationRequest) (*types.QueryAllApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var applications []types.Application
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	applicationStore := prefix.NewStore(store, types.KeyPrefix(types.ApplicationKeyPrefix))

	pageRes, err := query.Paginate(applicationStore, req.Pagination, func(key []byte, value []byte) error {
		var application types.Application
		if err := k.cdc.Unmarshal(value, &application); err != nil {
			return err
		}

		applications = append(applications, application)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllApplicationResponse{Application: applications, Pagination: pageRes}, nil
}

func (k Keeper) Application(goCtx context.Context, req *types.QueryGetApplicationRequest) (*types.QueryGetApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetApplication(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("application not found: address %s", req.Address))
	}

	return &types.QueryGetApplicationResponse{Application: val}, nil
}
