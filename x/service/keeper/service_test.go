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
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func createNservices(keeper *keeper.Keeper, ctx sdk.Context, n int) []sharedtypes.Service {
	items := make([]sharedtypes.Service, n)
	for i := range items {
		items[i].Id = fmt.Sprintf("srv%d", i)
		items[i].Name = fmt.Sprintf("srv%d", i)

		keeper.SetService(ctx, items[i])
	}
	return items
}

func TestServiceModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1nhmtqf4gcmpxu0p6e53hpgtwj0llmsqpxtumcf", moduleAddress.String())
}

func TestServiceGet(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	items := createNservices(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetService(ctx,
			item.Id,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestServiceRemove(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	items := createNservices(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveService(ctx,
			item.Id,
		)
		_, found := keeper.GetService(ctx,
			item.Id,
		)
		require.False(t, found)
	}
}

func TestServiceGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	items := createNservices(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllServices(ctx)),
	)
}
