package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/x/service/types"
)

// AllServices queries all services.
func (k Keeper) AllServices(ctx context.Context, req *service.QueryAllServicesRequest) (*service.QueryAllServicesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var services []shared.Service

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceStore := prefix.NewStore(store, types.KeyPrefix(types.ServiceKeyPrefix))

	pageRes, err := query.Paginate(serviceStore, req.Pagination, func(key []byte, value []byte) error {
		var service shared.Service
		if err := k.cdc.Unmarshal(value, &service); err != nil {
			return err
		}

		services = append(services, service)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &service.QueryAllServicesResponse{Service: services, Pagination: pageRes}, nil
}

// Service returns the requested service if it exists.
func (k Keeper) Service(ctx context.Context, req *service.QueryGetServiceRequest) (*service.QueryGetServiceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	foundService, found := k.GetService(ctx, req.Id)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &service.QueryGetServiceResponse{Service: foundService}, nil
}
