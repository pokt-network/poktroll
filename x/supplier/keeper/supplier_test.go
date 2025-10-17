package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

// createNSuppliers creates n suppliers and stores them in the keeper
func createNSuppliers(keeper keeper.Keeper, ctx context.Context, n int) []sharedtypes.Supplier {
	suppliers := make([]sharedtypes.Supplier, n)
	for i := range suppliers {
		supplier := &suppliers[i]
		supplier.OwnerAddress = sample.AccAddressBech32()
		supplier.OperatorAddress = sample.AccAddressBech32()
		supplier.Stake = &cosmostypes.Coin{Denom: "upokt", Amount: math.NewInt(int64(i))}
		serviceId := fmt.Sprintf("svc%d", i)
		supplier.Services = []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: serviceId,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     fmt.Sprintf("http://localhost:%d", i),
						RpcType: sharedtypes.RPCType_JSON_RPC,
						Configs: make([]*sharedtypes.ConfigOption, 0),
					},
				},
			},
		}
		supplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplier.OperatorAddress,
			supplier.Services,
			1,
			sharedtypes.NoDeactivationHeight,
		)
		supplier.ServiceUsageMetrics = map[string]*sharedtypes.ServiceUsageMetrics{
			serviceId: {ServiceId: serviceId},
		}
		keeper.SetAndIndexDehydratedSupplier(ctx, *supplier)
	}

	return suppliers
}

// DEV_NOTE: The account address is derived off of the module's semantic name (supplier).
// This test is a helper for us to easily identify the underlying address.
// See Module Accounts for more details: https://docs.cosmos.network/main/learn/beginner/accounts#module-accounts
func TestModuleAddressSupplier(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1j40dzzmn6cn9kxku7a5tjnud6hv37vesr5ccaa", moduleAddress.String())
}

func TestSupplier_Get(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 10)
	for _, supplier := range suppliers {
		supplierFound, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx,
			supplier.OperatorAddress,
		)
		require.True(t, isSupplierFound)
		require.Equal(t,
			nullify.Fill(&supplier),
			nullify.Fill(&supplierFound),
		)
	}
}

func TestSupplier_Remove(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 10)
	for _, supplier := range suppliers {
		supplierModuleKeepers.RemoveSupplier(ctx, supplier.OperatorAddress)
		_, isSupplierFound := supplierModuleKeepers.GetSupplier(ctx,
			supplier.OperatorAddress,
		)
		require.False(t, isSupplierFound)
	}
}

func TestSupplier_GetAll(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(suppliers),
		nullify.Fill(supplierModuleKeepers.GetAllSuppliers(ctx)),
	)
}

func TestSupplier_GetAllUnstakingSuppliersIterator(t *testing.T) {
	// Test configuration constants
	const (
		// Total number of suppliers to create
		totalSuppliers = 10

		// Range of suppliers to mark as unstaking (inclusive start, exclusive end)
		unstakingStartIdx = 2
		unstakingEndIdx   = 8

		// Height at which unstaking session ends
		unstakeSessionEndHeight = 100

		// Expected number of unstaking suppliers (derived from range)
		expectedUnstakingCount = unstakingEndIdx - unstakingStartIdx // 6
	)

	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)

	// Create suppliers
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, totalSuppliers)

	// Mark suppliers as unstaking
	for i := unstakingStartIdx; i < unstakingEndIdx; i++ {
		suppliers[i].UnstakeSessionEndHeight = unstakeSessionEndHeight
		supplierModuleKeepers.SetAndIndexDehydratedSupplier(ctx, suppliers[i])
	}

	// Get all unstaking suppliers
	iterator := supplierModuleKeepers.GetAllUnstakingSuppliersIterator(ctx)
	defer iterator.Close()

	// Count unstaking suppliers from iterator
	unstakingCount := 0
	for ; iterator.Valid(); iterator.Next() {
		unstakingCount++
	}

	// Verify we found expected number of unstaking suppliers
	require.Equal(t, expectedUnstakingCount, unstakingCount)
}

func TestServiceConfigUpdateIterators(t *testing.T) {
	// Test configuration constants
	const (
		// Total number of suppliers to create
		totalSuppliers = 100

		// Service identifier used for testing
		testServiceID = "service1"

		// Range of suppliers that will use testServiceID (inclusive start, exclusive end)
		service1StartIdx = 25
		service1EndIdx   = 75
		service1Count    = service1EndIdx - service1StartIdx // 50

		// Activation height configuration
		specificActivationHeight              = 10
		activationStartIdx                    = 10
		activationEndIdx                      = 35
		suppliersWithSpecificActivationHeight = activationEndIdx - activationStartIdx // 25

		// Deactivation height configuration
		specificDeactivationHeight              = 21
		deactivationEndIdx                      = 12                 // First 12 suppliers (0-11)
		suppliersWithSpecificDeactivationHeight = deactivationEndIdx // 12

		// Special supplier with multiple services
		multiServiceSupplierIdx = 99
		multiServiceCount       = 10

		// Default query height used in tests
		defaultQueryHeight = 1

		// Expected active service1 configs at query height 1
		// (service1Count minus suppliers that have activation height > query height)
		// The overlap between service1 suppliers and those with future activation height is 10
		expectedActiveService1Configs = service1Count - 10 // 40
	)

	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	keeper := *supplierModuleKeepers.Keeper

	// Create suppliers with service config updates
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, totalSuppliers)

	// Configure suppliers for testServiceID
	for i := service1StartIdx; i < service1EndIdx; i++ {
		suppliers[i].Services[0].ServiceId = testServiceID
		suppliers[i].ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			suppliers[i].OperatorAddress,
			suppliers[i].Services,
			defaultQueryHeight,
			sharedtypes.NoDeactivationHeight,
		)
	}

	// Configure suppliers with specific activation height
	for i := activationStartIdx; i < activationEndIdx; i++ {
		suppliers[i].ServiceConfigHistory[0].ActivationHeight = specificActivationHeight
	}

	// Configure suppliers with specific deactivation height
	for i := 0; i < deactivationEndIdx; i++ {
		suppliers[i].ServiceConfigHistory[0].DeactivationHeight = specificDeactivationHeight
	}

	// Configure a supplier with multiple service configs
	suppliers[multiServiceSupplierIdx].Services = make([]*sharedtypes.SupplierServiceConfig, multiServiceCount)
	for i := range multiServiceCount {
		suppliers[multiServiceSupplierIdx].Services[i] = &sharedtypes.SupplierServiceConfig{
			ServiceId: fmt.Sprintf("sup_svc_%d", i),
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     fmt.Sprintf("http://localhost:%d", i),
					RpcType: sharedtypes.RPCType_JSON_RPC,
					Configs: make([]*sharedtypes.ConfigOption, 0),
				},
			},
		}
	}
	suppliers[multiServiceSupplierIdx].ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		suppliers[multiServiceSupplierIdx].OperatorAddress,
		suppliers[multiServiceSupplierIdx].Services,
		defaultQueryHeight,
		sharedtypes.NoDeactivationHeight,
	)

	for _, supplier := range suppliers {
		keeper.SetAndIndexDehydratedSupplier(ctx, supplier)
	}

	t.Run("GetServiceConfigUpdatesIterator", func(t *testing.T) {
		// Test for testServiceID which should have expected number of active service config updates
		iterator := keeper.GetServiceConfigUpdatesIterator(ctx, testServiceID, defaultQueryHeight)
		defer iterator.Close()

		numConfigUpdatesWithService1 := 0
		for ; iterator.Valid(); iterator.Next() {
			config, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, testServiceID, config.Service.ServiceId)
			numConfigUpdatesWithService1++
		}
		require.Equal(t, expectedActiveService1Configs, numConfigUpdatesWithService1)
	})

	t.Run("GetActivatedServiceConfigUpdatesIterator", func(t *testing.T) {
		// Test for specific activation height
		iterator := keeper.GetActivatedServiceConfigUpdatesIterator(ctx, specificActivationHeight)
		defer iterator.Close()

		numConfigUpdatesWithActivationHeight := 0
		for ; iterator.Valid(); iterator.Next() {
			config, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, int64(specificActivationHeight), config.ActivationHeight)
			numConfigUpdatesWithActivationHeight++
		}

		require.Equal(t, suppliersWithSpecificActivationHeight, numConfigUpdatesWithActivationHeight)
	})

	t.Run("GetDeactivatedServiceConfigUpdatesIterator", func(t *testing.T) {
		// Test for specific deactivation height
		iterator := keeper.GetDeactivatedServiceConfigUpdatesIterator(ctx, specificDeactivationHeight)
		defer iterator.Close()

		numConfigUpdatesWithDeactivationHeight := 0
		for ; iterator.Valid(); iterator.Next() {
			config, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, int64(specificDeactivationHeight), config.DeactivationHeight)
			numConfigUpdatesWithDeactivationHeight++
		}

		require.Equal(t, suppliersWithSpecificDeactivationHeight, numConfigUpdatesWithDeactivationHeight)
	})
}

func TestSupplier_Query(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*keeper.Keeper, ctx, 2)

	tests := []struct {
		desc        string
		request     *types.QueryGetSupplierRequest
		response    *types.QueryGetSupplierResponse
		expectedErr error
	}{
		{
			desc: "supplier found",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: suppliers[0].OperatorAddress,
			},
			response: &types.QueryGetSupplierResponse{
				Supplier: suppliers[0],
			},
		},
		{
			desc: "supplier not found",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: "non_existent_address",
			},
			expectedErr: status.Error(codes.NotFound, fmt.Sprintf("supplier with address: %q", "non_existent_address")),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := keeper.Supplier(ctx, test.request)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.ErrorContains(t, stat.Err(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				require.Equal(t,
					nullify.Fill(test.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestSuppliers_QueryAll_Pagination(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*keeper.Keeper, ctx, 5)

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(suppliers); i += step {
			req := &types.QueryAllSuppliersRequest{
				Pagination: &query.PageRequest{
					Offset: uint64(i),
					Limit:  uint64(step),
				},
			}
			resp, err := keeper.AllSuppliers(ctx, req)
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
		var nextKey []byte
		for i := 0; i < len(suppliers); i += step {
			req := &types.QueryAllSuppliersRequest{
				Pagination: &query.PageRequest{
					Key:   nextKey,
					Limit: uint64(step),
				},
			}
			resp, err := keeper.AllSuppliers(ctx, req)
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(suppliers),
				nullify.Fill(resp.Supplier),
			)
			nextKey = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		req := &types.QueryAllSuppliersRequest{
			Pagination: &query.PageRequest{
				Offset:     0,
				Limit:      uint64(len(suppliers)),
				CountTotal: true,
			},
		}
		resp, err := keeper.AllSuppliers(ctx, req)
		require.NoError(t, err)
		require.Equal(t, len(suppliers), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(suppliers),
			nullify.Fill(resp.Supplier),
		)
	})
}

func TestSuppliers_QueryAll_Filters(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*keeper.Keeper, ctx, 10)
	ctx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(60)

	serviceId := "service1"

	// Update the first 7 suppliers to be staked for service1
	for i := range 7 {
		suppliers[i].Services[0].ServiceId = serviceId
		suppliers[i].ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			suppliers[i].OperatorAddress,
			suppliers[i].Services,
			1,
			sharedtypes.NoDeactivationHeight,
		)
	}

	// Make the 4th supplier has a past deactivation height for service1
	suppliers[3].ServiceConfigHistory[0].DeactivationHeight = 50

	// Make the 6th supplier has a future activation height for service1
	suppliers[5].ServiceConfigHistory[0].ActivationHeight = 100

	// Save all suppliers updates to the keeper
	for _, supplier := range suppliers {
		keeper.SetAndIndexDehydratedSupplier(ctx, supplier)
	}

	t.Run("Filter By ServiceId", func(t *testing.T) {
		// Assuming the first supplier has at least one service
		req := &types.QueryAllSuppliersRequest{
			Pagination: &query.PageRequest{
				Offset: 0,
				Limit:  uint64(len(suppliers)),
			},
			ServiceId: serviceId,
		}
		resp, err := keeper.AllSuppliers(ctx, req)
		require.NoError(t, err)

		// Verify each returned supplier has the specified service
		for _, s := range resp.Supplier {
			hasService := false
			for _, service := range s.Services {
				if service.ServiceId == serviceId {
					hasService = true
					break
				}
			}
			require.True(t, hasService, "Returned supplier does not have the specified service")
		}

		// Only 5 suppliers should be returned, as the 4th and 6th suppliers are not
		// active for service1
		require.Len(t, resp.Supplier, 5)
	})
}
