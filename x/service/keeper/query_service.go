package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ServiceAll(ctx context.Context, req *types.QueryAllServiceRequest) (*types.QueryAllServiceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var services []sharedtypes.Service

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceStore := prefix.NewStore(store, types.KeyPrefix(types.ServiceKeyPrefix))

	pageRes, err := query.Paginate(serviceStore, req.Pagination, func(key []byte, value []byte) error {
		var service sharedtypes.Service
		if err := k.cdc.Unmarshal(value, &service); err != nil {
			return err
		}

		services = append(services, service)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllServiceResponse{Service: services, Pagination: pageRes}, nil
}

func (k Keeper) Service(ctx context.Context, req *types.QueryGetServiceRequest) (*types.QueryGetServiceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	service, found := k.GetService(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetServiceResponse{Service: service}, nil
}
