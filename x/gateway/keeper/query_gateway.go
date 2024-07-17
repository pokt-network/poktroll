package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k Keeper) AllGateways(ctx context.Context, req *gateway.QueryAllGatewaysRequest) (*gateway.QueryAllGatewaysResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var gateways []gateway.Gateway

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	gatewayStore := prefix.NewStore(store, types.KeyPrefix(types.GatewayKeyPrefix))

	pageRes, err := query.Paginate(gatewayStore, req.Pagination, func(key []byte, value []byte) error {
		var gw gateway.Gateway
		if err := k.cdc.Unmarshal(value, &gw); err != nil {
			return err
		}

		gateways = append(gateways, gw)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &gateway.QueryAllGatewaysResponse{Gateways: gateways, Pagination: pageRes}, nil
}

func (k Keeper) Gateway(ctx context.Context, req *gateway.QueryGetGatewayRequest) (*gateway.QueryGetGatewayResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	foundGateway, found := k.GetGateway(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("gateway not found: address %s", req.Address))
	}
	return &gateway.QueryGetGatewayResponse{Gateway: foundGateway}, nil
}
