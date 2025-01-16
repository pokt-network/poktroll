package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

// The module address is derived off of its semantic name.
// This test is a helper for us to easily identify the underlying address.
func TestModuleAddressSupplier(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1j40dzzmn6cn9kxku7a5tjnud6hv37vesr5ccaa", moduleAddress.String())
}

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

func TestSupplierQuery(t *testing.T) {
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
				require.ErrorIs(t, stat.Err(), test.expectedErr)
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

func TestSupplierGet(t *testing.T) {
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

func TestSupplierRemove(t *testing.T) {
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

func TestSupplierGetAll(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(suppliers),
		nullify.Fill(supplierModuleKeepers.GetAllSuppliers(ctx)),
	)
}
