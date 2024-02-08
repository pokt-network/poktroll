package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k Keeper) SupplierAll(ctx context.Context, req *types.QueryAllSupplierRequest) (*types.QueryAllSupplierResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var suppliers []sharedtypes.Supplier

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	supplierStore := prefix.NewStore(store, types.KeyPrefix(types.SupplierKeyPrefix))

	pageRes, err := query.Paginate(supplierStore, req.Pagination, func(key []byte, value []byte) error {
		var supplier sharedtypes.Supplier
		if err := k.cdc.Unmarshal(value, &supplier); err != nil {
			return err
		}

		suppliers = append(suppliers, supplier)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSupplierResponse{Supplier: suppliers, Pagination: pageRes}, nil
}

func (k Keeper) Supplier(ctx context.Context, req *types.QueryGetSupplierRequest) (*types.QueryGetSupplierResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, found := k.GetSupplier(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetSupplierResponse{Supplier: val}, nil
}
