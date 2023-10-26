package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"pocket/cmd/pocketd/cmd"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/testutil/sample"
	"pocket/x/application/keeper"
	"pocket/x/application/types"
	sharedtypes "pocket/x/shared/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func createNApplication(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Application {
	apps := make([]types.Application, n)
	for i, app := range apps {
		app.Address = sample.AccAddress()
		app.Stake = &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(int64(i))}
		app.ServiceConfigs = []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: &sharedtypes.ServiceId{Id: fmt.Sprintf("svc%d", i)},
			},
		}
		keeper.SetApplication(ctx, app)
	}
	return apps
}

func TestApplicationGet(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplication(keeper, ctx, 10)
	for _, app := range apps {
		appFound, isAppFound := keeper.GetApplication(ctx,
			app.Address,
		)
		require.True(t, isAppFound)
		require.Equal(t,
			nullify.Fill(&app),
			nullify.Fill(&appFound),
		)
	}
}
func TestApplicationRemove(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplication(keeper, ctx, 10)
	for _, app := range apps {
		keeper.RemoveApplication(ctx,
			app.Address,
		)
		_, found := keeper.GetApplication(ctx,
			app.Address,
		)
		require.False(t, found)
	}
}

func TestApplicationGetAll(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplication(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(apps),
		nullify.Fill(keeper.GetAllApplication(ctx)),
	)
}

// The application module address is derived off of its semantic name.
// This test is a helper for us to easily identify the underlying address.
func TestApplicationModuleAddress(t *testing.T) {
	moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
	require.Equal(t, "pokt1rl3gjgzexmplmds3tq3r3yk84zlwdl6djzgsvm", moduleAddress.String())
}
