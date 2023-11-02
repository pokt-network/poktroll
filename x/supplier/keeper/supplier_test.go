package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
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

// The application module address is derived off of its semantic name.
// This test is a helper for us to easily identify the underlying address.
func TestApplicationModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1j40dzzmn6cn9kxku7a5tjnud6hv37vesr5ccaa", moduleAddress.String())
}
