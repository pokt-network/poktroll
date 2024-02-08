package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/pokt-network/poktroll/x/gateway/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GatewayAll(ctx context.Context, req *types.QueryAllGatewayRequest) (*types.QueryAllGatewayResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var gateways []types.Gateway

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	gatewayStore := prefix.NewStore(store, types.KeyPrefix(types.GatewayKeyPrefix))

	pageRes, err := query.Paginate(gatewayStore, req.Pagination, func(key []byte, value []byte) error {
		var gateway types.Gateway
		if err := k.cdc.Unmarshal(value, &gateway); err != nil {
			return err
		}

		gateways = append(gateways, gateway)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGatewayResponse{Gateway: gateways, Pagination: pageRes}, nil
}

func (k Keeper) Gateway(ctx context.Context, req *types.QueryGetGatewayRequest) (*types.QueryGetGatewayResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetGateway(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("gateway not found: address %s", req.Address))

	}

	return &types.QueryGetGatewayResponse{Gateway: val}, nil
}
