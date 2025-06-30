package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// AllSuppliers returns a paginated list of all suppliers in the store.
// If a serviceId is provided, it filters suppliers to only those staking for that service.
// If an ownerAddress is provided, it filters suppliers to only those owned by that address.
// The returned suppliers are fully hydrated with their service configurations and history.
func (k Keeper) AllSuppliers(
	ctx context.Context,
	req *types.QueryAllSuppliersRequest,
) (*types.QueryAllSuppliersResponse, error) {
	logger := k.Logger().With("method", "AllSuppliers")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.GetServiceId() != "" {
		return k.getAllServiceSuppliers(ctx, logger, req)
	} else if req.GetOwnerAddress() != "" {
		return k.getAllOwnerSuppliers(ctx, logger, req)
	} else {
		return k.getAllSuppliers(ctx, logger, req)
	}
}

// Supplier retrieves a specific supplier by operator address.
// The returned supplier can be hydrated or dehydrated based on the request flag.
func (k Keeper) Supplier(
	ctx context.Context,
	req *types.QueryGetSupplierRequest,
) (*types.QueryGetSupplierResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// Retrieve a dehydrated supplier first
	supplier, found := k.GetDehydratedSupplier(ctx, req.OperatorAddress)
	if !found {
		err := fmt.Sprintf("supplier with operator address: %q", req.GetOperatorAddress())
		return nil, status.Error(
			codes.NotFound,
			types.ErrSupplierNotFound.Wrap(err).Error(),
		)
	}

	// Conditionally hydrate supplier fields based on dehydrated flag
	if req.GetDehydrated() {
		k.hydratePartialDehydratedSupplierServiceConfigs(ctx, &supplier)
	} else {
		k.hydrateFullSupplierServiceConfigs(ctx, &supplier)
	}

	return &types.QueryGetSupplierResponse{Supplier: supplier}, nil
}

// getAllSuppliers retrieves all suppliers from the store with pagination support.
// Each supplier's service configurations are fully hydrated before being returned.
func (k Keeper) getAllSuppliers(
	ctx context.Context,
	logger log.Logger,
	req *types.QueryAllSuppliersRequest,
) (*types.QueryAllSuppliersResponse, error) {
	supplierStore := k.getSupplierStore(ctx)

	var suppliers []sharedtypes.Supplier

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

			// Conditionally hydrate supplier fields based on dehydrated flag
			if req.GetDehydrated() {
				k.hydratePartialDehydratedSupplierServiceConfigs(ctx, &supplier)
			} else {
				k.hydrateFullSupplierServiceConfigs(ctx, &supplier)
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

// getAllServiceSuppliers retrieves all suppliers that are staked for specific service.
// Only returns suppliers with active service configurations at the current block height.
func (k Keeper) getAllServiceSuppliers(
	ctx context.Context,
	logger log.Logger,
	req *types.QueryAllSuppliersRequest,
) (*types.QueryAllSuppliersResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Prepare supplier stores
	serviceConfigUpdateStore := k.getServiceConfigUpdatesByServiceStore(ctx, req.GetServiceId())
	supplierStore := k.getSupplierStore(ctx)

	// Initialize a slice to collect suppliers
	var suppliers []sharedtypes.Supplier
	// Initialize a map to track which suppliers have been processed to avoid
	// duplicate suppliers in the results
	selectedSuppliersMap := make(map[string]struct{})

	// Iterate over all service config updates for the specified service
	pageRes, err := query.Paginate(
		serviceConfigUpdateStore,
		req.Pagination,
		func(key []byte, serviceConfigUpdateBz []byte) error {
			// Unmarshal the service config update from the store
			var serviceConfigUpdate sharedtypes.ServiceConfigUpdate
			if err := k.cdc.Unmarshal(serviceConfigUpdateBz, &serviceConfigUpdate); err != nil {
				err = fmt.Errorf("unmarshaling service config update with key (hex): %x: %+v", key, err)
				logger.Error(err.Error())
				return status.Error(codes.Internal, err.Error())
			}

			// Skip service configurations that are not active at the current block height
			if !serviceConfigUpdate.IsActive(currentHeight) {
				return nil
			}

			// Skip suppliers that have already been added to the results
			if _, ok := selectedSuppliersMap[serviceConfigUpdate.OperatorAddress]; ok {
				return nil
			}
			selectedSuppliersMap[serviceConfigUpdate.OperatorAddress] = struct{}{}

			// Retrieve the supplier data using the operator address from the service config
			supplierKey := types.SupplierOperatorKey(serviceConfigUpdate.OperatorAddress)
			supplierBz := supplierStore.Get(supplierKey)
			var supplier sharedtypes.Supplier
			if err := k.cdc.Unmarshal(supplierBz, &supplier); err != nil {
				err = fmt.Errorf("unmarshaling supplier with key (hex): %x: %+v", key, err)
				logger.Error(err.Error())
				return status.Error(codes.Internal, err.Error())
			}

			// Conditionally load service configurations and history into the supplier object
			if req.GetDehydrated() {
				k.hydratePartialDehydratedSupplierServiceConfigs(ctx, &supplier)
			} else {
				k.hydrateFullSupplierServiceConfigs(ctx, &supplier)
			}

			// Add the supplier to the results
			suppliers = append(suppliers, supplier)
			return nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSuppliersResponse{Supplier: suppliers, Pagination: pageRes}, nil
}

// getAllOwnerSuppliers retrieves all suppliers that are owned by a specific address.
// It iterates through all suppliers and filters by owner address.
func (k Keeper) getAllOwnerSuppliers(
	ctx context.Context,
	logger log.Logger,
	req *types.QueryAllSuppliersRequest,
) (*types.QueryAllSuppliersResponse, error) {
	supplierStore := k.getSupplierStore(ctx)

	var suppliers []sharedtypes.Supplier

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

			// Filter by owner address
			if supplier.OwnerAddress != req.GetOwnerAddress() {
				return nil
			}

			// Conditionally hydrate supplier fields based on dehydrated flag
			if req.GetDehydrated() {
				k.hydratePartialDehydratedSupplierServiceConfigs(ctx, &supplier)
			} else {
				k.hydrateFullSupplierServiceConfigs(ctx, &supplier)
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