package keeper_test

import (
	"context"
	"strconv"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/proto/types/application"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNApplications(keeper keeper.Keeper, ctx context.Context, n int) []application.Application {
	apps := make([]application.Application, n)
	for i := range apps {
		apps[i].Address = strconv.Itoa(i)
		// Setting pending undelegations since nullify.Fill() does not seem to do it.
		apps[i].PendingUndelegations = make(map[uint64]application.UndelegatingGatewayList)

		keeper.SetApplication(ctx, apps[i])
	}
	return apps
}

func init() {
	cmd.InitSDKConfig()
}

// The module address is derived off of its semantic name.
// This test is a helper for us to easily identify the underlying address.
func TestModuleAddressApplication(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1rl3gjgzexmplmds3tq3r3yk84zlwdl6djzgsvm", moduleAddress.String())
}

func TestApplicationGet(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	for _, app := range apps {
		foundApp, found := keeper.GetApplication(ctx, app.Address)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&app),
			nullify.Fill(&foundApp),
		)
	}
}
func TestApplicationRemove(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	for _, app := range apps {
		keeper.RemoveApplication(ctx, app.Address)
		_, found := keeper.GetApplication(ctx, app.Address)
		require.False(t, found)
	}
}

func TestApplicationGetAll(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(apps),
		nullify.Fill(keeper.GetAllApplications(ctx)),
	)
}
