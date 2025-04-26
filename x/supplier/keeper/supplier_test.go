package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		supplier.OwnerAddress = sample.AccAddress()
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
		supplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplier.OperatorAddress,
			supplier.Services,
			1,
			sharedtest.NoDeactivationHeight,
		)
		keeper.SetSupplier(ctx, *supplier)
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
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)

	// Create 6 suppliers with unstaking height
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 10)
	for i := 2; i < 8; i++ {
		suppliers[i].UnstakeSessionEndHeight = 100
		supplierModuleKeepers.SetSupplier(ctx, suppliers[i])
	}

	// Get all unstaking suppliers
	iterator := supplierModuleKeepers.GetAllUnstakingSuppliersIterator(ctx)
	defer iterator.Close()

	// Count unstaking suppliers from iterator
	unstakingCount := 0
	for ; iterator.Valid(); iterator.Next() {
		unstakingCount++
	}

	// Verify we found exactly 6 unstaking suppliers
	require.Equal(t, 6, unstakingCount)
}

func TestServiceConfigUpdateIterators(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	keeper := *supplierModuleKeepers.Keeper

	// Create 100 suppliers with service config updates
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 100)

	// 50 of them will be for service1
	for i := 25; i < 75; i++ {
		suppliers[i].Services[0].ServiceId = "service1"
		suppliers[i].ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			suppliers[i].OperatorAddress,
			suppliers[i].Services,
			1,
			sharedtest.NoDeactivationHeight,
		)
	}

	// 25 will have an activation height of 10
	for i := 10; i < 35; i++ {
		suppliers[i].ServiceConfigHistory[0].ActivationHeight = 10
	}

	// 12 will have a deactivation height of 21
	for i := 0; i < 12; i++ {
		suppliers[i].ServiceConfigHistory[0].DeactivationHeight = 21
	}

	// Supplier 100 will have 10 service configs
	suppliers[99].Services = make([]*sharedtypes.SupplierServiceConfig, 10)
	for i := range 10 {
		suppliers[99].Services[i] = &sharedtypes.SupplierServiceConfig{
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
	suppliers[99].ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
		suppliers[99].OperatorAddress,
		suppliers[99].Services,
		1,
		sharedtest.NoDeactivationHeight,
	)

	for _, supplier := range suppliers {
		keeper.SetSupplier(ctx, supplier)
	}

	t.Run("GetServiceConfigUpdatesIterator", func(t *testing.T) {
		// Test for service1 which should have 50 service config updates
		iterator := keeper.GetServiceConfigUpdatesIterator(ctx, "service1")
		defer iterator.Close()

		numConfigUpdatesWithService1 := 0
		for ; iterator.Valid(); iterator.Next() {
			config, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, "service1", config.Service.ServiceId)
			numConfigUpdatesWithService1++
		}
		require.Equal(t, 50, numConfigUpdatesWithService1)
	})

	t.Run("GetActivatedServiceConfigUpdatesIterator", func(t *testing.T) {
		// Test for activation height 10 which should have 25 service config updates
		iterator := keeper.GetActivatedServiceConfigUpdatesIterator(ctx, 10)
		defer iterator.Close()

		numConfigUpdatesWithActivationHeigh10 := 0
		for ; iterator.Valid(); iterator.Next() {
			config, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, int64(10), config.ActivationHeight)
			numConfigUpdatesWithActivationHeigh10++
		}

		require.Equal(t, 25, numConfigUpdatesWithActivationHeigh10)
	})

	t.Run("GetDeactivatedServiceConfigUpdatesIterator", func(t *testing.T) {
		// Test for deactivation height 21 which should have 12 service config updates
		iterator := keeper.GetDeactivatedServiceConfigUpdatesIterator(ctx, 21)
		defer iterator.Close()

		numConfigUpdatesWithDeactivationHeight21 := 0
		for ; iterator.Valid(); iterator.Next() {
			config, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, int64(21), config.DeactivationHeight)
			numConfigUpdatesWithDeactivationHeight21++
		}

		require.Equal(t, 12, numConfigUpdatesWithDeactivationHeight21)
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
			sharedtest.NoDeactivationHeight,
		)
	}

	// Make the 4th supplier has a past deactivation height for service1
	suppliers[3].ServiceConfigHistory[0].DeactivationHeight = 50

	// Make the 6th supplier has a future activation height for service1
	suppliers[5].ServiceConfigHistory[0].ActivationHeight = 100

	// Save all suppliers updates to the keeper
	for _, supplier := range suppliers {
		keeper.SetSupplier(ctx, supplier)
	}

	t.Run("Filter By ServiceId", func(t *testing.T) {
		// Assuming the first supplier has at least one service
		req := &types.QueryAllSuppliersRequest{
			Pagination: &query.PageRequest{
				Offset: 0,
				Limit:  uint64(len(suppliers)),
			},
			Filter: &types.QueryAllSuppliersRequest_ServiceId{
				ServiceId: serviceId,
			},
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
