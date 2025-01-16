package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k Keeper) AllSuppliers(
	ctx context.Context,
	req *types.QueryAllSuppliersRequest,
) (*types.QueryAllSuppliersResponse, error) {
	logger := k.Logger().With("method", "AllSuppliers")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var suppliers []sharedtypes.Supplier

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	supplierStore := prefix.NewStore(store, types.KeyPrefix(types.SupplierKeyOperatorPrefix))

	pageRes, err := query.Paginate(
		supplierStore,
		req.Pagination,
		func(key []byte, value []byte) error {
			var supplier sharedtypes.Supplier
			if err := k.cdc.Unmarshal(value, &supplier); err != nil {
				err = fmt.Errorf("unmarshaling supplier with key (hex): %x: %+v", key, err)
				logger.Error(err.Error())
				return status.Error(codes.Internal, err.Error())
			}

			suppliers = append(suppliers, supplier)
			return nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSuppliersResponse{Supplier: suppliers, Pagination: pageRes}, nil
}

func (k Keeper) Supplier(
	ctx context.Context,
	req *types.QueryGetSupplierRequest,
) (*types.QueryGetSupplierResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	supplier, found := k.GetSupplier(ctx, req.OperatorAddress)
	if !found {
		// TODO_TECHDEBT(@bryanchriswhite, #384): conform to logging conventions once established
		msg := fmt.Sprintf("supplier with address %q", req.GetOperatorAddress())
		return nil, status.Error(codes.NotFound, msg)
	}

	return &types.QueryGetSupplierResponse{Supplier: supplier}, nil
}
