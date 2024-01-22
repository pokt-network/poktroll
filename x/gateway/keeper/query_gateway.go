package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// GatewayAll retrieves all gateway from the store handling the query request.
func (k Keeper) GatewayAll(
	goCtx context.Context,
	req *types.QueryAllGatewayRequest,
) (*types.QueryAllGatewayResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var gateways []types.Gateway
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
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

// Gateway retrieves the specified gateway from the store handling the query request.
func (k Keeper) Gateway(
	goCtx context.Context,
	req *types.QueryGetGatewayRequest,
) (*types.QueryGetGatewayResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetGateway(
		ctx,
		req.Address,
	)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("gateway not found: address %s", req.Address))
	}

	return &types.QueryGetGatewayResponse{Gateway: val}, nil
}
