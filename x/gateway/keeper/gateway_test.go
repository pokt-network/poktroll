package keeper_test

import (
	"strconv"
	"testing"

	"pocket/cmd/pocketd/cmd"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/x/gateway/keeper"
	"pocket/x/gateway/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func createNGateway(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Gateway {
	items := make([]types.Gateway, n)
	for i := range items {
		items[i].Address = strconv.Itoa(i)

		keeper.SetGateway(ctx, items[i])
	}
	return items
}

func TestGatewayModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1f6j7u6875p2cvyrgjr0d2uecyzah0kget9vlpl", moduleAddress.String())
}

func TestGatewayGet(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	items := createNGateway(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetGateway(ctx,
			item.Address,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestGatewayRemove(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	items := createNGateway(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveGateway(ctx,
			item.Address,
		)
		_, found := keeper.GetGateway(ctx,
			item.Address,
		)
		require.False(t, found)
	}
}

func TestGatewayGetAll(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	items := createNGateway(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllGateway(ctx)),
	)
}
