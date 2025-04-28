package keeper_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

// testAppModifier represents any function that can be used to modify an application being instantiated for testing purposes.
type testAppModifier func(app *types.Application)

func createNApplications(keeper keeper.Keeper, ctx context.Context, n int, testAppModifiers ...testAppModifier) []types.Application {
	apps := make([]types.Application, n)
	for i := range apps {
		apps[i].Address = strconv.Itoa(i)
		// Setting pending undelegations since nullify.Fill() does not seem to do it.
		apps[i].PendingUndelegations = make(map[uint64]types.UndelegatingGatewayList)

		for _, modifier := range testAppModifiers {
			modifier(&apps[i])
		}

		keeper.SetApplication(ctx, apps[i])
	}
	return apps
}

// testAppModifierDelegateeAddr adds the supplied gateway address to the application's delegatee list if the application's address matches
// the supplied address list.
func withAppDelegateeGatewayAddr(delegateeGatewayAddr string, appsWithDelegationAddr []string) testAppModifier {
	return func(app *types.Application) {
		if slices.Contains(appsWithDelegationAddr, app.Address) {
			app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, delegateeGatewayAddr)
		}
	}
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
		keeper.RemoveApplication(ctx, app)
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

func TestApplicationGetAllIterator(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)
	apps := createNApplications(keeper, ctx, 10)
	allAppsIterator := keeper.GetAllApplicationsIterator(ctx)
	defer allAppsIterator.Close()

	retrievedApps := make([]types.Application, 0)
	for ; allAppsIterator.Valid(); allAppsIterator.Next() {
		app, err := allAppsIterator.Value()
		require.NoError(t, err)
		retrievedApps = append(retrievedApps, app)
	}

	require.ElementsMatch(t,
		nullify.Fill(apps),
		nullify.Fill(retrievedApps),
	)
}

func TestApplication_GetAllUnstakingApplicationsIterator(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)

	// Create 10 applications, 6 with unstaking height
	apps := createNApplications(keeper, ctx, 10)
	for i := 2; i < 8; i++ {
		apps[i].UnstakeSessionEndHeight = 100
		keeper.SetApplication(ctx, apps[i])
	}

	// Get all unstaking applications
	iterator := keeper.GetAllUnstakingApplicationsIterator(ctx)
	defer iterator.Close()

	// Count unstaking applications from iterator
	unstakingCount := 0
	unstakingApps := make([]types.Application, 0)
	for ; iterator.Valid(); iterator.Next() {
		app, err := iterator.Value()
		require.NoError(t, err)
		unstakingApps = append(unstakingApps, app)
		unstakingCount++
	}

	// Verify we found exactly 6 unstaking applications
	require.Equal(t, 6, unstakingCount)

	// Verify each application has the correct unstaking height
	for _, app := range unstakingApps {
		require.Equal(t, uint64(100), app.UnstakeSessionEndHeight)
	}
}

func TestApplication_GetAllTransferringApplicationsIterator(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)

	// Create 10 applications, 5 with pending transfers
	apps := createNApplications(keeper, ctx, 10)
	for i := 3; i < 8; i++ {
		apps[i].PendingTransfer = &types.PendingApplicationTransfer{
			DestinationAddress: sample.AccAddress(),
			SessionEndHeight:   100,
		}
		keeper.SetApplication(ctx, apps[i])
	}

	// Get all transferring applications
	iterator := keeper.GetAllTransferringApplicationsIterator(ctx)
	defer iterator.Close()

	// Count transferring applications from iterator
	transferringCount := 0
	transferringApps := make([]types.Application, 0)
	for ; iterator.Valid(); iterator.Next() {
		app, err := iterator.Value()
		require.NoError(t, err)
		transferringApps = append(transferringApps, app)
		transferringCount++
	}

	// Verify we found exactly 5 transferring applications
	require.Equal(t, 5, transferringCount)

	// Verify each application has the correct transfer pending height
	for _, app := range transferringApps {
		require.Equal(t, uint64(100), app.PendingTransfer.SessionEndHeight)
	}
}

func TestApplication_GetDelegationsIterator(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)

	// Create a gateway address that some applications will delegate to
	targetGatewayAddr := sample.AccAddress()

	// Create 10 applications, with 4 delegating to our test gateway
	delegateeGatewayAddrs := make([]string, 4)
	for i := 0; i < 4; i++ {
		delegateeGatewayAddrs[i] = sample.AccAddress()
	}
	apps := createNApplications(keeper, ctx, 10)

	// Make all apps delegate to the delegateeGatewayAddrs
	for _, app := range apps {
		app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, delegateeGatewayAddrs...)
		keeper.SetApplication(ctx, app)
	}

	// delegate app 3 and 5 to the target gateway
	apps[3].DelegateeGatewayAddresses = append(apps[3].DelegateeGatewayAddresses, targetGatewayAddr)
	apps[5].DelegateeGatewayAddresses = append(apps[5].DelegateeGatewayAddresses, targetGatewayAddr)
	keeper.SetApplication(ctx, apps[3])
	keeper.SetApplication(ctx, apps[5])

	// Get applications delegating to the target gateway
	iterator := keeper.GetDelegationsIterator(ctx, targetGatewayAddr)
	defer iterator.Close()

	// Count delegating applications from iterator
	delegatingCount := 0
	delegatingApps := make([]types.Application, 0)
	for ; iterator.Valid(); iterator.Next() {
		app, err := iterator.Value()
		require.NoError(t, err)
		delegatingApps = append(delegatingApps, app)
		delegatingCount++
	}

	// Verify we found exactly 2 delegating applications
	require.Equal(t, 2, delegatingCount)

	// Verify each application has the gateway address in its delegatee list
	for _, app := range delegatingApps {
		require.Contains(t, app.DelegateeGatewayAddresses, targetGatewayAddr)
	}

	// Verify the addresses match what we expect
	expectedAddresses := []string{apps[3].Address, apps[5].Address}
	require.ElementsMatch(t, expectedAddresses, []string{delegatingApps[0].Address, delegatingApps[1].Address})
}

func TestApplication_GetUndelegationsIterator(t *testing.T) {
	keeper, ctx := keepertest.ApplicationKeeper(t)

	// Create gateway addresses for undelegations
	gateway1 := sample.AccAddress()
	gateway2 := sample.AccAddress()
	gateway3 := sample.AccAddress()

	// Create 5 applications with various undelegations
	apps := createNApplications(keeper, ctx, 5)

	// Set up undelegations for app 0
	height100Undelegations := types.UndelegatingGatewayList{
		GatewayAddresses: []string{gateway3},
	}
	apps[0].PendingUndelegations = map[uint64]types.UndelegatingGatewayList{
		100: height100Undelegations,
	}

	// Set up undelegations for app 2
	height150Undelegations := types.UndelegatingGatewayList{
		GatewayAddresses: []string{gateway2},
	}
	apps[2].PendingUndelegations = map[uint64]types.UndelegatingGatewayList{
		150: height150Undelegations,
	}

	// Set up undelegations for app 4 with multiple heights and gateways
	height200Undelegations := types.UndelegatingGatewayList{
		GatewayAddresses: []string{gateway1, gateway2},
	}
	apps[4].PendingUndelegations = map[uint64]types.UndelegatingGatewayList{
		100: height100Undelegations,
		200: height200Undelegations,
	}

	// Save all applications
	for _, app := range apps {
		keeper.SetApplication(ctx, app)
	}

	t.Run("GetUndelegationsForSpecificApplication", func(t *testing.T) {
		// Get undelegations for app 0
		iterator := keeper.GetUndelegationsIterator(ctx, "0")
		defer iterator.Close()

		undelegationCount := 0
		for ; iterator.Valid(); iterator.Next() {
			undelegation, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, "0", undelegation.ApplicationAddress)
			require.Equal(t, gateway3, undelegation.GatewayAddress)
			undelegationCount++
		}

		require.Equal(t, 1, undelegationCount)
	})

	t.Run("GetUndelegationsForApplicationWithMultipleGateways", func(t *testing.T) {
		// Get undelegations for app 4 which has 2 gateways
		iterator := keeper.GetUndelegationsIterator(ctx, "4")
		defer iterator.Close()

		undelegationCount := 0
		gatewayAddrs := make([]string, 0)
		for ; iterator.Valid(); iterator.Next() {
			undelegation, err := iterator.Value()
			require.NoError(t, err)
			require.Equal(t, "4", undelegation.ApplicationAddress)
			gatewayAddrs = append(gatewayAddrs, undelegation.GatewayAddress)
			undelegationCount++
		}

		require.Equal(t, 3, undelegationCount)
		require.ElementsMatch(t, []string{gateway1, gateway2, gateway3}, gatewayAddrs)
	})

	t.Run("GetAllUndelegations", func(t *testing.T) {
		// Get all undelegations across applications
		iterator := keeper.GetUndelegationsIterator(ctx, appkeeper.ALL_UNDELEGATIONS)
		defer iterator.Close()

		// Should have 4 undelegations total across all applications
		undelegationCount := 0
		for ; iterator.Valid(); iterator.Next() {
			undelegation, err := iterator.Value()
			require.NoError(t, err)

			// Verify undelegation has correct application address
			appAddr := undelegation.ApplicationAddress
			require.Contains(t, []string{"0", "2", "4"}, appAddr)

			// Verify gateway address is one of our test gateways
			gatewayAddr := undelegation.GatewayAddress
			require.Contains(t, []string{gateway1, gateway2, gateway3}, gatewayAddr)

			undelegationCount++
		}

		require.Equal(t, 5, undelegationCount)
	})
}
