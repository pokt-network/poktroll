package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func createNServices(keeper *keeper.Keeper, ctx sdk.Context, n int) []sharedtypes.Service {
	services := make([]sharedtypes.Service, n)
	for i := range services {
		services[i].Id = fmt.Sprintf("srvId%d", i)
		services[i].Name = fmt.Sprintf("svcName%d", i)

		keeper.SetService(ctx, services[i])
	}
	return services
}

func TestServiceModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1nhmtqf4gcmpxu0p6e53hpgtwj0llmsqpxtumcf", moduleAddress.String())
}

func TestServiceGet(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	services := createNServices(keeper, ctx, 10)
	for _, service := range services {
		service, found := keeper.GetService(ctx,
			service.Id,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&service),
			nullify.Fill(&service),
		)
	}
}

func TestServiceRemove(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	services := createNServices(keeper, ctx, 10)
	for _, service := range services {
		keeper.RemoveService(ctx,
			service.Id,
		)
		_, found := keeper.GetService(ctx,
			service.Id,
		)
		require.False(t, found)
	}
}

func TestServiceGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	services := createNServices(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(services),
		nullify.Fill(keeper.GetAllServices(ctx)),
	)
}
