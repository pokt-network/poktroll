package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/testutil/sample"
	sharedtypes "pocket/x/shared/types"
	"pocket/x/supplier/keeper"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNSupplier(keeper *keeper.Keeper, ctx sdk.Context, n int) []sharedtypes.Supplier {
	suppliers := make([]sharedtypes.Supplier, n)
	for i := range suppliers {
		supplier := &suppliers[i]
		supplier.Address = sample.AccAddress()
		supplier.Stake = &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(int64(i))}
		supplier.Services = []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: fmt.Sprintf("svc%d", i)},
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

func TestSupplierGet(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSupplier(keeper, ctx, 10)
	for _, supplier := range suppliers {
		supplierFound, isSupplierFound := keeper.GetSupplier(ctx,
			supplier.Address,
		)
		require.True(t, isSupplierFound)
		require.Equal(t,
			nullify.Fill(&supplier),
			nullify.Fill(&supplierFound),
		)
	}
}
func TestSupplierRemove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSupplier(keeper, ctx, 10)
	for _, supplier := range suppliers {
		keeper.RemoveSupplier(ctx,
			supplier.Address,
		)
		_, isSupplierFound := keeper.GetSupplier(ctx,
			supplier.Address,
		)
		require.False(t, isSupplierFound)
	}
}

func TestSupplierGetAll(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSupplier(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(suppliers),
		nullify.Fill(keeper.GetAllSupplier(ctx)),
	)
}
