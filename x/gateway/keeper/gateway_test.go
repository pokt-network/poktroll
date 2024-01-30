package keeper_test

import (
	"context"
	"strconv"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func createNGateway(keeper keeper.Keeper, ctx context.Context, n int) []types.Gateway {
	gateways := make([]types.Gateway, n)
	for i := range gateways {
		gateways[i].Address = strconv.Itoa(i)

		keeper.SetGateway(ctx, gateways[i])
	}
	return gateways
}

func TestGatewayModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1f6j7u6875p2cvyrgjr0d2uecyzah0kget9vlpl", moduleAddress.String())
}

func TestGatewayGet(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	gateways := createNGateway(keeper, ctx, 10)
	for _, item := range gateways {
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
	gateways := createNGateway(keeper, ctx, 10)
	for _, gateway := range gateways {
		keeper.RemoveGateway(ctx,
			gateway.Address,
		)
		_, found := keeper.GetGateway(ctx,
			gateway.Address,
		)
		require.False(t, found)
	}
}

func TestGatewayGetAll(t *testing.T) {
	keeper, ctx := keepertest.GatewayKeeper(t)
	gateways := createNGateway(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(gateways),
		nullify.Fill(keeper.GetAllGateway(ctx)),
	)
}
