package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/pokt-network/pocket/cmd/pocketd/cmd"
	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
	"github.com/pokt-network/pocket/x/supplier/keeper"
	"github.com/pokt-network/pocket/x/supplier/types"
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
	suppliers := createNSuppliers(*keeper.Keeper, ctx, 5)

	t.Run("Filter By ServiceId", func(t *testing.T) {
		// Assuming the first supplier has at least one service
		serviceId := suppliers[0].Services[0].ServiceId
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
	})
}
