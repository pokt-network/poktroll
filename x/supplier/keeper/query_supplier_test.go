package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestSupplierQuerySingle(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 2)
	supplierAddr := sample.AccAddress()

	tests := []struct {
		desc        string
		request     *types.QueryGetSupplierRequest
		response    *types.QueryGetSupplierResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: suppliers[0].OperatorAddress,
			},
			response: &types.QueryGetSupplierResponse{Supplier: suppliers[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: suppliers[1].OperatorAddress,
			},
			response: &types.QueryGetSupplierResponse{Supplier: suppliers[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: supplierAddr,
			},
			expectedErr: status.Error(
				codes.NotFound,
				types.ErrSupplierNotFound.Wrapf(
					"supplier with operator address: \"%s\"", supplierAddr,
				).Error(),
			),
		},
		{
			desc:        "InvalidRequest",
			expectedErr: status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := supplierModuleKeepers.Supplier(ctx, test.request)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(test.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestSupplierQueryPaginated(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSuppliersRequest {
		return &types.QueryAllSuppliersRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(suppliers); i += step {
			resp, err := supplierModuleKeepers.AllSuppliers(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(suppliers),
				nullify.Fill(resp.Supplier),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(suppliers); i += step {
			resp, err := supplierModuleKeepers.AllSuppliers(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(suppliers),
				nullify.Fill(resp.Supplier),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(suppliers), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(suppliers),
			nullify.Fill(resp.Supplier),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := supplierModuleKeepers.AllSuppliers(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestSupplierQueryFilterByServiceId(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 5)

	// Get the first service ID from the first supplier to use as filter
	firstServiceId := suppliers[0].Services[0].ServiceId

	request := &types.QueryAllSuppliersRequest{
		Filter: &types.QueryAllSuppliersRequest_ServiceId{
			ServiceId: firstServiceId,
		},
		Pagination: &query.PageRequest{
			Limit: uint64(len(suppliers)),
		},
	}

	resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
	require.NoError(t, err)

	// createNSuppliers assigns a separate service to each supplier
	// so we can only expect one supplier to have the filtered service.
	require.Len(t, resp.Supplier, 1)

	// Verify each returned supplier has the filtered service
	for _, supplier := range resp.Supplier {
		hasService := false
		for _, service := range supplier.Services {
			if service.ServiceId == firstServiceId {
				hasService = true
				break
			}
		}
		require.True(t, hasService, "Supplier should have the filtered service")
	}
}

func TestSupplierQueryDehydrated(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 3)

	t.Run("AllSuppliers_Dehydrated", func(t *testing.T) {
		request := &types.QueryAllSuppliersRequest{
			Dehydrated: true,
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, len(suppliers))

		// Verify each returned supplier is dehydrated
		for _, supplier := range resp.Supplier {
			// Should not have service config history
			require.Nil(t, supplier.ServiceConfigHistory, "Dehydrated supplier should not have service config history")

			// Should have services but without rev_share
			require.NotNil(t, supplier.Services, "Dehydrated supplier should still have services")
			for _, service := range supplier.Services {
				require.Nil(t, service.RevShare, "Dehydrated supplier services should not have rev_share")
				// Should still have other fields like service_id and endpoints
				require.NotEmpty(t, service.ServiceId, "Service should still have service_id")
				require.NotNil(t, service.Endpoints, "Service should still have endpoints")
			}
		}
	})

	t.Run("AllSuppliers_Hydrated", func(t *testing.T) {
		request := &types.QueryAllSuppliersRequest{
			Dehydrated: false,
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, len(suppliers))

		// Verify each returned supplier is fully hydrated
		for _, supplier := range resp.Supplier {
			// Should have service config history
			require.NotNil(t, supplier.ServiceConfigHistory, "Hydrated supplier should have service config history")
			require.NotEmpty(t, supplier.ServiceConfigHistory, "Hydrated supplier should have non-empty service config history")

			// Should have services
			require.NotNil(t, supplier.Services, "Hydrated supplier should have services")
			require.NotEmpty(t, supplier.Services, "Hydrated supplier should have non-empty services")

			// Note: RevShare may be nil in test data, so we don't require it to be present
			// The key difference is that dehydrated mode explicitly sets RevShare to nil
		}
	})

	t.Run("AllSuppliers_FilterByServiceId_Dehydrated", func(t *testing.T) {
		// Get the first service ID from the first supplier to use as filter
		firstServiceId := suppliers[0].Services[0].ServiceId

		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_ServiceId{
				ServiceId: firstServiceId,
			},
			Dehydrated: true,
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)

		// Should find at least one supplier with the filtered service
		require.GreaterOrEqual(t, len(resp.Supplier), 1)

		// Verify each returned supplier is dehydrated and has the filtered service
		for _, supplier := range resp.Supplier {
			// Should not have service config history
			require.Nil(t, supplier.ServiceConfigHistory, "Dehydrated supplier should not have service config history")

			// Should have services but without rev_share
			require.NotNil(t, supplier.Services, "Dehydrated supplier should still have services")
			require.GreaterOrEqual(t, len(supplier.Services), 1)

			// Verify the supplier has the filtered service
			hasService := false
			for _, service := range supplier.Services {
				require.Nil(t, service.RevShare, "Dehydrated supplier services should not have rev_share")
				if service.ServiceId == firstServiceId {
					hasService = true
				}
			}
			require.True(t, hasService, "Supplier should have the filtered service")
		}
	})

	t.Run("AllSuppliers_Dehydrated_WithRevShare", func(t *testing.T) {
		// Create a supplier with RevShare data to better test dehydration
		supplierWithRevShare := suppliers[0]
		supplierWithRevShare.Services[0].RevShare = []*sharedtypes.ServiceRevenueShare{
			{
				Address:            sample.AccAddress(),
				RevSharePercentage: 50,
			},
			{
				Address:            sample.AccAddress(),
				RevSharePercentage: 50,
			},
		}
		supplierWithRevShare.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierWithRevShare.OperatorAddress,
			supplierWithRevShare.Services,
			1,
			sharedtypes.NoDeactivationHeight,
		)
		supplierModuleKeepers.SetAndIndexDehydratedSupplier(ctx, supplierWithRevShare)

		// Test dehydrated query
		request := &types.QueryAllSuppliersRequest{
			Dehydrated: true,
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, len(suppliers))

		var testSupplierDehydrated sharedtypes.Supplier
		for _, supplier := range resp.Supplier {
			if supplier.OperatorAddress == supplierWithRevShare.OperatorAddress {
				testSupplierDehydrated = supplier
				break
			}
		}
		require.Nil(t, testSupplierDehydrated.ServiceConfigHistory, "Dehydrated supplier should not have service config history")
		require.NotNil(t, testSupplierDehydrated.Services, "Dehydrated supplier should still have services")
		require.Len(t, testSupplierDehydrated.Services, 1)
		require.Nil(t, testSupplierDehydrated.Services[0].RevShare, "Dehydrated supplier services should not have rev_share")

		// Test hydrated query for comparison
		request.Dehydrated = false
		resp, err = supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, len(suppliers))

		var testSupplierHydrated sharedtypes.Supplier
		for _, supplier := range resp.Supplier {
			if supplier.OperatorAddress == supplierWithRevShare.OperatorAddress {
				testSupplierHydrated = supplier
				break
			}
		}
		require.NotNil(t, testSupplierHydrated.ServiceConfigHistory, "Hydrated supplier should have service config history")
		require.NotEmpty(t, testSupplierHydrated.ServiceConfigHistory, "Hydrated supplier should have non-empty service config history")
		require.NotNil(t, testSupplierHydrated.Services, "Hydrated supplier should have services")
		require.Len(t, testSupplierHydrated.Services, 1)
		require.NotNil(t, testSupplierHydrated.Services[0].RevShare, "Hydrated supplier services should have rev_share")
		require.Len(t, testSupplierHydrated.Services[0].RevShare, 2, "Should have 2 rev_share entries")
	})
}

func TestSupplierShowDehydrated(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 1)
	supplierOperatorAddr := suppliers[0].OperatorAddress

	t.Run("ShowSupplier_Dehydrated", func(t *testing.T) {
		request := &types.QueryGetSupplierRequest{
			OperatorAddress: supplierOperatorAddr,
			Dehydrated:      true,
		}

		resp, err := supplierModuleKeepers.Supplier(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, resp)

		supplier := resp.Supplier
		// Should not have service config history
		require.Nil(t, supplier.ServiceConfigHistory, "Dehydrated supplier should not have service config history")

		// Should have services but without rev_share
		require.NotNil(t, supplier.Services, "Dehydrated supplier should still have services")
		for _, service := range supplier.Services {
			require.Nil(t, service.RevShare, "Dehydrated supplier services should not have rev_share")
			// Should still have other fields like service_id and endpoints
			require.NotEmpty(t, service.ServiceId, "Service should still have service_id")
			require.NotNil(t, service.Endpoints, "Service should still have endpoints")
		}
	})

	t.Run("ShowSupplier_Hydrated", func(t *testing.T) {
		request := &types.QueryGetSupplierRequest{
			OperatorAddress: supplierOperatorAddr,
			Dehydrated:      false,
		}

		resp, err := supplierModuleKeepers.Supplier(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, resp)

		supplier := resp.Supplier
		// Should have service config history
		require.NotNil(t, supplier.ServiceConfigHistory, "Hydrated supplier should have service config history")
		require.NotEmpty(t, supplier.ServiceConfigHistory, "Hydrated supplier should have non-empty service config history")

		// Should have services
		require.NotNil(t, supplier.Services, "Hydrated supplier should have services")

		// Note: RevShare may be nil in test data, so we don't require it to be present
		// The key difference is that dehydrated mode explicitly sets RevShare to nil
	})

	t.Run("ShowSupplier_Dehydrated_WithRevShare", func(t *testing.T) {
		// Create a supplier with RevShare data to better test dehydration
		supplierWithRevShare := suppliers[0]
		supplierWithRevShare.Services[0].RevShare = []*sharedtypes.ServiceRevenueShare{
			{
				Address:            sample.AccAddress(),
				RevSharePercentage: 30,
			},
			{
				Address:            sample.AccAddress(),
				RevSharePercentage: 70,
			},
		}
		supplierWithRevShare.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierWithRevShare.OperatorAddress,
			supplierWithRevShare.Services,
			1,
			sharedtypes.NoDeactivationHeight,
		)
		supplierModuleKeepers.SetAndIndexDehydratedSupplier(ctx, supplierWithRevShare)

		// Test dehydrated query
		request := &types.QueryGetSupplierRequest{
			OperatorAddress: supplierWithRevShare.OperatorAddress,
			Dehydrated:      true,
		}

		resp, err := supplierModuleKeepers.Supplier(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, resp)

		supplier := resp.Supplier
		require.Nil(t, supplier.ServiceConfigHistory, "Dehydrated supplier should not have service config history")
		require.NotNil(t, supplier.Services, "Dehydrated supplier should still have services")
		require.Len(t, supplier.Services, 1)
		require.Nil(t, supplier.Services[0].RevShare, "Dehydrated supplier services should not have rev_share")

		// Test hydrated query for comparison
		request.Dehydrated = false
		resp, err = supplierModuleKeepers.Supplier(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, resp)

		supplier = resp.Supplier
		require.NotNil(t, supplier.ServiceConfigHistory, "Hydrated supplier should have service config history")
		require.NotEmpty(t, supplier.ServiceConfigHistory, "Hydrated supplier should have non-empty service config history")
		require.NotNil(t, supplier.Services, "Hydrated supplier should have services")
		require.Len(t, supplier.Services, 1)
		require.NotNil(t, supplier.Services[0].RevShare, "Hydrated supplier services should have rev_share")
		require.Len(t, supplier.Services[0].RevShare, 2, "Should have 2 rev_share entries")
	})
}

func TestSupplierQueryFilterByOwnerAddress(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	
	// Create suppliers with specific owners
	ownerAddr1 := sample.AccAddress()
	ownerAddr2 := sample.AccAddress()
	ownerAddr3 := sample.AccAddress()
	
	// Create suppliers with different owners
	suppliers := make([]sharedtypes.Supplier, 5)
	for i := range suppliers {
		supplier := &suppliers[i]
		supplier.OperatorAddress = sample.AccAddress()
		supplier.Stake = &sdk.Coin{Denom: "upokt", Amount: math.NewInt(int64(i))}
		supplier.Services = []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: fmt.Sprintf("svc%d", i),
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     fmt.Sprintf("http://localhost:%d", i),
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
		}
		
		// Assign owners in a specific pattern
		switch i {
		case 0, 1:
			supplier.OwnerAddress = ownerAddr1 // 2 suppliers owned by ownerAddr1
		case 2, 3:
			supplier.OwnerAddress = ownerAddr2 // 2 suppliers owned by ownerAddr2
		case 4:
			supplier.OwnerAddress = ownerAddr3 // 1 supplier owned by ownerAddr3
		}
		
		// Create service config history for the supplier
		supplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplier.OperatorAddress,
			supplier.Services,
			1,
			sharedtypes.NoDeactivationHeight,
		)
		
		// Store and index the supplier
		supplierModuleKeepers.SetAndIndexDehydratedSupplier(ctx, *supplier)
	}

	t.Run("FilterByOwnerAddress_MultipleSuppliers", func(t *testing.T) {
		// Test filtering by ownerAddr1 (should return 2 suppliers)
		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_OwnerAddress{
				OwnerAddress: ownerAddr1,
			},
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 2, "Should return exactly 2 suppliers owned by ownerAddr1")

		// Verify each returned supplier has the correct owner
		for _, supplier := range resp.Supplier {
			require.Equal(t, ownerAddr1, supplier.OwnerAddress, "Supplier should be owned by ownerAddr1")
		}
	})

	t.Run("FilterByOwnerAddress_SingleSupplier", func(t *testing.T) {
		// Test filtering by ownerAddr3 (should return 1 supplier)
		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_OwnerAddress{
				OwnerAddress: ownerAddr3,
			},
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 1, "Should return exactly 1 supplier owned by ownerAddr3")

		// Verify the returned supplier has the correct owner
		require.Equal(t, ownerAddr3, resp.Supplier[0].OwnerAddress, "Supplier should be owned by ownerAddr3")
	})

	t.Run("FilterByOwnerAddress_NoResults", func(t *testing.T) {
		// Test filtering by an address that doesn't own any suppliers
		nonOwnerAddr := sample.AccAddress()
		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_OwnerAddress{
				OwnerAddress: nonOwnerAddr,
			},
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 0, "Should return no suppliers for non-owner address")
	})

	t.Run("FilterByOwnerAddress_WithPagination", func(t *testing.T) {
		// Test pagination with owner filtering (ownerAddr1 has 2 suppliers)
		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_OwnerAddress{
				OwnerAddress: ownerAddr1,
			},
			Pagination: &query.PageRequest{
				Limit: 1, // Only get 1 supplier per page
			},
		}

		// First page
		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 1, "First page should return 1 supplier")
		require.Equal(t, ownerAddr1, resp.Supplier[0].OwnerAddress, "Supplier should be owned by ownerAddr1")
		require.NotNil(t, resp.Pagination.NextKey, "Should have next page")

		// Second page
		request.Pagination.Key = resp.Pagination.NextKey
		resp, err = supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 1, "Second page should return 1 supplier")
		require.Equal(t, ownerAddr1, resp.Supplier[0].OwnerAddress, "Supplier should be owned by ownerAddr1")
	})

	t.Run("FilterByOwnerAddress_Dehydrated", func(t *testing.T) {
		// Add RevShare to one of the suppliers to test dehydration
		supplierWithRevShare := suppliers[0]
		supplierWithRevShare.Services[0].RevShare = []*sharedtypes.ServiceRevenueShare{
			{
				Address:            sample.AccAddress(),
				RevSharePercentage: 25,
			},
			{
				Address:            sample.AccAddress(),
				RevSharePercentage: 75,
			},
		}
		supplierWithRevShare.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierWithRevShare.OperatorAddress,
			supplierWithRevShare.Services,
			1,
			sharedtypes.NoDeactivationHeight,
		)
		supplierModuleKeepers.SetAndIndexDehydratedSupplier(ctx, supplierWithRevShare)

		// Test dehydrated query with owner filter
		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_OwnerAddress{
				OwnerAddress: ownerAddr1,
			},
			Dehydrated: true,
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 2, "Should return 2 dehydrated suppliers owned by ownerAddr1")

		// Verify each returned supplier is dehydrated and has the correct owner
		for _, supplier := range resp.Supplier {
			require.Equal(t, ownerAddr1, supplier.OwnerAddress, "Supplier should be owned by ownerAddr1")
			require.Nil(t, supplier.ServiceConfigHistory, "Dehydrated supplier should not have service config history")
			require.NotNil(t, supplier.Services, "Dehydrated supplier should still have services")
			for _, service := range supplier.Services {
				require.Nil(t, service.RevShare, "Dehydrated supplier services should not have rev_share")
			}
		}
	})

	t.Run("FilterByOwnerAddress_Hydrated", func(t *testing.T) {
		// Test hydrated query with owner filter
		request := &types.QueryAllSuppliersRequest{
			Filter: &types.QueryAllSuppliersRequest_OwnerAddress{
				OwnerAddress: ownerAddr2,
			},
			Dehydrated: false,
			Pagination: &query.PageRequest{
				Limit: uint64(len(suppliers)),
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 2, "Should return 2 hydrated suppliers owned by ownerAddr2")

		// Verify each returned supplier is hydrated and has the correct owner
		for _, supplier := range resp.Supplier {
			require.Equal(t, ownerAddr2, supplier.OwnerAddress, "Supplier should be owned by ownerAddr2")
			require.NotNil(t, supplier.ServiceConfigHistory, "Hydrated supplier should have service config history")
			require.NotEmpty(t, supplier.ServiceConfigHistory, "Hydrated supplier should have non-empty service config history")
			require.NotNil(t, supplier.Services, "Hydrated supplier should have services")
		}
	})
}
