package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AllServices queries all services.
func (k Keeper) AllServices(ctx context.Context, req *types.QueryAllServicesRequest) (*types.QueryAllServicesResponse, error) {
	logger := k.Logger().With("method", "AllServices")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var services []sharedtypes.Service

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceStore := prefix.NewStore(store, types.KeyPrefix(types.ServiceKeyPrefix))

	pageRes, err := query.Paginate(serviceStore, req.Pagination, func(key []byte, value []byte) error {
		var service sharedtypes.Service
		if err := k.cdc.Unmarshal(value, &service); err != nil {
			logger.Error(fmt.Sprintf("unable to unmarshal service with key (hex): %x: %+v", key, err))
			return status.Error(codes.Internal, err.Error())
		}

		services = append(services, service)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllServicesResponse{Service: services, Pagination: pageRes}, nil
}

// Service returns the requested service if it exists.
func (k Keeper) Service(ctx context.Context, req *types.QueryGetServiceRequest) (*types.QueryGetServiceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	service, found := k.GetService(ctx, req.Id)
	if !found {
		return nil, status.Error(codes.NotFound, "service ID not found")
	}

	return &types.QueryGetServiceResponse{Service: service}, nil
}
