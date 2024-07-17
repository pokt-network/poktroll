package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/x/application/types"
)

func (k Keeper) AllApplications(
	ctx context.Context,
	req *application.QueryAllApplicationsRequest,
) (*application.QueryAllApplicationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var apps []application.Application

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	applicationStore := prefix.NewStore(store, types.KeyPrefix(types.ApplicationKeyPrefix))

	pageRes, err := query.Paginate(applicationStore, req.Pagination, func(key []byte, value []byte) error {
		var app application.Application
		if err := k.cdc.Unmarshal(value, &app); err != nil {
			return err
		}

		// Ensure that the PendingUndelegations is an empty map and not nil when
		// unmarshalling an app that has no pending undelegations.
		if app.PendingUndelegations == nil {
			app.PendingUndelegations = make(map[uint64]application.UndelegatingGatewayList)
		}

		apps = append(apps, app)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &application.QueryAllApplicationsResponse{Applications: apps, Pagination: pageRes}, nil
}

func (k Keeper) Application(ctx context.Context, req *application.QueryGetApplicationRequest) (*application.QueryGetApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	app, found := k.GetApplication(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "application not found")
	}

	return &application.QueryGetApplicationResponse{Application: app}, nil
}
