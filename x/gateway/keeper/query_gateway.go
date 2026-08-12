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

		// Strip the card when the caller asked for a dehydrated response.
		//
		// This matters most here: a card can be up to MaxServiceMetadataSizeBytes (256 KiB),
		// so a default page of 100 carried cards would be ~25 MB and exceed the 4 MB default
		// gRPC message limit. Callers enumerating gateways almost never want the payloads;
		// `query gateway card <address>` fetches an individual one.
		if req.Dehydrated {
			gateway.Metadata = nil
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
		return nil, status.Error(codes.NotFound, fmt.Sprintf("gateway not found: address %s", req.Address))
	}

	// Strip the card when the caller asked for a dehydrated response.
	// Mirrors the same flag on the service queries. Defaults to false (card included).
	if req.Dehydrated {
		gateway.Metadata = nil
	}

	return &types.QueryGetGatewayResponse{Gateway: gateway}, nil
}
