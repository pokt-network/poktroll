package keeper

import (
	"context"
	"fmt"
	"slices"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/application/types"
)

// AllApplications returns all applications filtered based on any criteria specified in the request.
func (k Keeper) AllApplications(
	ctx context.Context,
	req *types.QueryAllApplicationsRequest,
) (*types.QueryAllApplicationsResponse, error) {
	logger := k.Logger().With("method", "AllApplications")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var apps []types.Application

	// TODO_IMPROVE(#767): Consider adding a custom onchain index (similar to proofs)
	// based on other parameters (e.g. serviceId) if/when the performance of the
	// flags used to filter the response becomes an issue.
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	applicationStore := prefix.NewStore(store, types.KeyPrefix(types.ApplicationKeyPrefix))

	pageRes, err := query.Paginate(applicationStore, req.Pagination, func(key []byte, value []byte) error {
		var application types.Application
		if err := k.cdc.Unmarshal(value, &application); err != nil {
			logger.Error(fmt.Sprintf("unmarshaling application with key (hex): %x: %+v", key, err))
			return status.Error(codes.Internal, err.Error())
		}

		// Filter out the application if the request specifies a gateway address delegated to as a constraint
		gatewayDelegatedToAddrFilter := req.GetGatewayAddressDelegatedTo()
		if gatewayDelegatedToAddrFilter != "" {
			if !slices.Contains(application.DelegateeGatewayAddresses, req.GatewayAddressDelegatedTo) {
				return nil
			}
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
