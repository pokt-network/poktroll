package keeper_test

import (
	"strconv"
	"testing"

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
				Limit: 1,
			},
		}

		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 1)

		supplier := resp.Supplier[0]
		require.Nil(t, supplier.ServiceConfigHistory, "Dehydrated supplier should not have service config history")
		require.NotNil(t, supplier.Services, "Dehydrated supplier should still have services")
		require.Len(t, supplier.Services, 1)
		require.Nil(t, supplier.Services[0].RevShare, "Dehydrated supplier services should not have rev_share")

		// Test hydrated query for comparison
		request.Dehydrated = false
		resp, err = supplierModuleKeepers.AllSuppliers(ctx, request)
		require.NoError(t, err)
		require.Len(t, resp.Supplier, 1)

		supplier = resp.Supplier[0]
		require.NotNil(t, supplier.ServiceConfigHistory, "Hydrated supplier should have service config history")
		require.NotEmpty(t, supplier.ServiceConfigHistory, "Hydrated supplier should have non-empty service config history")
		require.NotNil(t, supplier.Services, "Hydrated supplier should have services")
		require.Len(t, supplier.Services, 1)
		require.NotNil(t, supplier.Services[0].RevShare, "Hydrated supplier services should have rev_share")
		require.Len(t, supplier.Services[0].RevShare, 2, "Should have 2 rev_share entries")
	})
}
