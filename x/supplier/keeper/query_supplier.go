package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/supplier"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k Keeper) AllSuppliers(
	ctx context.Context,
	req *supplier.QueryAllSuppliersRequest,
) (*supplier.QueryAllSuppliersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var suppliers []shared.Supplier

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	supplierStore := prefix.NewStore(store, types.KeyPrefix(types.SupplierKeyPrefix))

	pageRes, err := query.Paginate(
		supplierStore,
		req.Pagination,
		func(key []byte, value []byte) error {
			var supplier shared.Supplier
			if err := k.cdc.Unmarshal(value, &supplier); err != nil {
				return err
			}

			suppliers = append(suppliers, supplier)
			return nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &supplier.QueryAllSuppliersResponse{Supplier: suppliers, Pagination: pageRes}, nil
}

func (k Keeper) Supplier(
	ctx context.Context,
	req *supplier.QueryGetSupplierRequest,
) (*supplier.QueryGetSupplierResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	foundSupplier, found := k.GetSupplier(ctx, req.Address)
	if !found {
		// TODO_TECHDEBT(@bryanchriswhite, #384): conform to logging conventions once established
		msg := fmt.Sprintf("supplier with address %q", req.GetAddress())
		return nil, status.Error(codes.NotFound, msg)
	}

	return &supplier.QueryGetSupplierResponse{Supplier: foundSupplier}, nil
}
