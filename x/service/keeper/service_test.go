package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/cmd/pocketd/cmd"
	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/x/service/keeper"
	"github.com/pokt-network/pocket/x/service/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func createNServices(keeper keeper.Keeper, ctx context.Context, n int) []sharedtypes.Service {
	services := make([]sharedtypes.Service, n)
	for i := range services {
		services[i].Id = strconv.Itoa(i)
		services[i].Name = fmt.Sprintf("svcName%d", i)

		keeper.SetService(ctx, services[i])
	}
	return services
}

// The module address is derived off of its semantic name.
// This test is a helper for us to easily identify the underlying address.
func TestModuleAddressService(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1nhmtqf4gcmpxu0p6e53hpgtwj0llmsqpxtumcf", moduleAddress.String())
}

func TestServiceGet(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	services := createNServices(keeper, ctx, 10)
	for _, service := range services {
		foundService, found := keeper.GetService(ctx, service.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&service),
			nullify.Fill(&foundService),
		)
	}
}
func TestServiceRemove(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
	services := createNServices(keeper, ctx, 10)
	for _, service := range services {
		keeper.RemoveService(ctx, service.Id)
		_, found := keeper.GetService(ctx, service.Id)
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
