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
)

// ComputeUnitsPerRelayAtHeight returns the compute_units_per_relay (cupr) that was
// effective at the given block height for a service. The RelayMiner uses this at
// session start to stamp relays, and claim validation uses the same value so the two
// always agree.
func (k Keeper) ComputeUnitsPerRelayAtHeight(
	ctx context.Context,
	req *types.QueryComputeUnitsPerRelayAtHeightRequest,
) (*types.QueryComputeUnitsPerRelayAtHeightResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service ID is required")
	}

	if req.BlockHeight < 0 {
		return nil, status.Error(codes.InvalidArgument, "block height must be non-negative")
	}

	_, serviceFound := k.GetService(ctx, req.ServiceId)
	if !serviceFound {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("serviceID: %s", req.ServiceId).Error(),
		)
	}

	computeUnitsPerRelay, _ := k.GetServiceComputeUnitsPerRelayAtHeight(ctx, req.ServiceId, req.BlockHeight)

	return &types.QueryComputeUnitsPerRelayAtHeightResponse{
		ComputeUnitsPerRelay: computeUnitsPerRelay,
	}, nil
}

// ComputeUnitsPerRelayHistory returns the history of cupr changes for a service.
func (k Keeper) ComputeUnitsPerRelayHistory(
	ctx context.Context,
	req *types.QueryComputeUnitsPerRelayHistoryRequest,
) (*types.QueryComputeUnitsPerRelayHistoryResponse, error) {
	logger := k.Logger().With("method", "ComputeUnitsPerRelayHistory")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service ID is required")
	}

	_, serviceFound := k.GetService(ctx, req.ServiceId)
	if !serviceFound {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("serviceID: %s", req.ServiceId).Error(),
		)
	}

	var history []types.ServiceComputeUnitsPerRelayUpdate

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	serviceHistoryPrefix := types.ServiceComputeUnitsPerRelayHistoryKeyPrefixForService(req.ServiceId)
	historyStore := prefix.NewStore(store, serviceHistoryPrefix)

	pageRes, err := query.Paginate(historyStore, req.Pagination, func(key []byte, value []byte) error {
		var update types.ServiceComputeUnitsPerRelayUpdate
		if err := k.cdc.Unmarshal(value, &update); err != nil {
			err = fmt.Errorf("unable to unmarshal ServiceComputeUnitsPerRelayUpdate with key (hex): %x: %w", key, err)
			logger.Error(err.Error())
			return status.Error(codes.Internal, err.Error())
		}

		history = append(history, update)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryComputeUnitsPerRelayHistoryResponse{
		ComputeUnitsPerRelayHistory: history,
		Pagination:                  pageRes,
	}, nil
}
