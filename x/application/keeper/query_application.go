package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/application/types"
)

func (k Keeper) AllApplications(ctx context.Context, req *types.QueryAllApplicationsRequest) (*types.QueryAllApplicationsResponse, error) {
	logger := k.Logger().With("method", "AllApplications")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var apps []types.Application

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	applicationStore := prefix.NewStore(store, types.KeyPrefix(types.ApplicationKeyPrefix))

	pageRes, err := query.Paginate(applicationStore, req.Pagination, func(key []byte, value []byte) error {
		var application types.Application
		if err := k.cdc.Unmarshal(value, &application); err != nil {
			logger.Error(fmt.Sprintf("unmarshaling application with key (hex): %x: %+v", key, err))
			return status.Error(codes.Internal, err.Error())
		}

		// Ensure that the PendingUndelegations is an empty map and not nil when
		// unmarshalling an app that has no pending undelegations.
		if application.PendingUndelegations == nil {
			application.PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)
		}

		apps = append(apps, application)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllApplicationsResponse{Applications: apps, Pagination: pageRes}, nil
}

func (k Keeper) Application(ctx context.Context, req *types.QueryGetApplicationRequest) (*types.QueryGetApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	app, found := k.GetApplication(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "application not found")
	}

	return &types.QueryGetApplicationResponse{Application: app}, nil
}
