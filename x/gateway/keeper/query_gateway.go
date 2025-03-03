package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k Keeper) AllGateways(ctx context.Context, req *types.QueryAllGatewaysRequest) (*types.QueryAllGatewaysResponse, error) {
	logger := k.Logger().With("method", "AllGateways")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var gateways []types.Gateway

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	gatewayStore := prefix.NewStore(store, types.KeyPrefix(types.GatewayKeyPrefix))

	pageRes, err := query.Paginate(gatewayStore, req.Pagination, func(key []byte, value []byte) error {
		var gateway types.Gateway
		if err := k.cdc.Unmarshal(value, &gateway); err != nil {
			logger.Error(fmt.Sprintf("unmarshaling gateway with key (hex): %x: %+v", key, err))
			return status.Error(codes.Internal, err.Error())
		}

		gateways = append(gateways, gateway)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGatewaysResponse{Gateways: gateways, Pagination: pageRes}, nil
}

func (k Keeper) Gateway(ctx context.Context, req *types.QueryGetGatewayRequest) (*types.QueryGetGatewayResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	gateway, found := k.GetGateway(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			types.ErrGatewayNotFound.Wrapf(
				"gateway with address: %s", req.Address,
			).Error(),
		)
	}
	return &types.QueryGetGatewayResponse{Gateway: gateway}, nil
}
