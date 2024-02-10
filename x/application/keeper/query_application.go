package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pokt-network/poktroll/x/application/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ApplicationAll(ctx context.Context, req *types.QueryAllApplicationRequest) (*types.QueryAllApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var applications []types.Application

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
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

func (k Keeper) Application(ctx context.Context, req *types.QueryGetApplicationRequest) (*types.QueryGetApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetApplication(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetApplicationResponse{Application: val}, nil
}
