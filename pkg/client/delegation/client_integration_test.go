//go:build integration

package delegation_test

// TODO(@h5law): Figure out how to use real components of the localnet
//	- Create app and gateway actors
//  - Stake them
//  - Delegate to the gateway
//  - Undelegate from the gateway
// Currently this test doesn't work, because (I think) it is using a mock
// keeper etc and this isnt actually interacting with the localnet where
// the DelegationClient is listening for events from.

import (
	"context"
	"sync"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	delegationIntegrationSubTimeout = 15 * time.Second
)

// TestDelegationClient_DelegateeChangesObservable tests that the DelegationClient
// can subscribe to the DelegateeChange events and that the events contain
// the correct AppAddress, it does so by simulating the delegation
// and undelegation of two applications to a gateway.
// TODO_TEST: This test needs to use real actors and not mocked ones for it to work
func TestDelegationClient_DelegateeChangesObservables(t *testing.T) {
	t.SkipNow() // TODO: remove once the test is working

	k, sdkCtx := keepertest.ApplicationKeeper(t)
	srv := appkeeper.NewMsgServerImpl(*k)
	ctx := sdk.WrapSDKContext(sdkCtx)

	// Generate an address for the mock gateway and mock stake it
	gatewayAddr := sample.AccAddress()
	keepertest.AddGatewayToStakedGatewayMap(t, gatewayAddr)

	// Cretae and stake two applications
	appAddr1 := prepareAppAndStake(t, ctx, sdkCtx, *k, srv)
	appAddr2 := prepareAppAndStake(t, ctx, sdkCtx, *k, srv)

	// Create the delegation client
	delegationClient := testdelegation.NewLocalnetClient(ctx, t)
	require.NotNil(t, delegationClient)
	t.Cleanup(func() {
		delegationClient.Close()
	})

	// Subscribe to the delegation events
	delegationSub := delegationClient.DelegateeChangesSequence(ctx).Subscribe(ctx)

	var (
		delegationMu            = sync.Mutex{}        // mutext to protect delegationChangeCounter
		delegationChangeCounter int                   // counter to keep track of the number of delegation changes
		expectedChanges         = 4                   // expected number of delegation changes
		errCh                   = make(chan error, 1) // channel to signal the test to stop
	)
	go func() {
		// The test will delegate from app1 to gateway, then from app2 to gateway
		// and then undelegate app1 from gateway and then undelegate app2 from gateway
		// We expect to receive 4 delegation changes where the address of the
		// DelegateeChange event alternates between app1 and app2
		var previousDelegateeChange client.DelegateeChange
		for change := range delegationSub.Ch() {
			// Verify that the DelegateeChange event is valid and that the address
			// of the DelegateeChange event alternates between app1 and app2
			if previousDelegateeChange != nil {
				require.NotEqual(t, previousDelegateeChange.AppAddress(), change.AppAddress())
				if previousDelegateeChange.AppAddress() == appAddr1 {
					require.Equal(t, appAddr2, change.AppAddress())
				} else {
					require.Equal(t, appAddr1, change.AppAddress())
				}
			}
			previousDelegateeChange = change

			require.NotEmpty(t, change)
			delegationMu.Lock()
			delegationChangeCounter++
			if delegationChangeCounter >= expectedChanges {
				errCh <- nil
				return
			}
			delegationMu.Unlock()
		}
	}()

	// Do the delegations and undelegations
	delegateAppToGateway(t, ctx, sdkCtx, *k, srv, appAddr1, gatewayAddr)
	delegateAppToGateway(t, ctx, sdkCtx, *k, srv, appAddr2, gatewayAddr)
	undelegateAppFromGateway(t, ctx, sdkCtx, *k, srv, appAddr1, gatewayAddr)
	undelegateAppFromGateway(t, ctx, sdkCtx, *k, srv, appAddr2, gatewayAddr)

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equal(t, expectedChanges, delegationChangeCounter)
	case <-time.After(delegationIntegrationSubTimeout):
		t.Fatalf(
			"timed out waiting for delegation subscription; expected %d delegation events, got %d",
			expectedChanges, delegationChangeCounter,
		)
	}
}

// prepareAppAndStake prepares an application and stakes it making sure that
// the application stakes successfully and exists in the application store.
// It returns the application address.
func prepareAppAndStake(
	t *testing.T,
	ctx context.Context,
	wctx sdk.Context,
	keeper appkeeper.Keeper,
	srv apptypes.MsgServer,
) (appAddress string) {
	t.Helper()
	// Generate an address for the application
	appAddr := sample.AccAddress()

	// Prepare the stake message
	stakeMsg := &apptypes.MsgStakeApplication{
		Address: appAddr,
		Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
		Services: []*sharedtypes.ApplicationServiceConfig{
			{
				Service: &sharedtypes.Service{Id: "svc1"},
			},
		},
	}

	// Stake the application & verify that the application exists
	_, err := srv.StakeApplication(ctx, stakeMsg)
	require.NoError(t, err)
	_, isAppFound := keeper.GetApplication(wctx, appAddr)
	require.True(t, isAppFound)

	return appAddr
}

func delegateAppToGateway(
	t *testing.T,
	ctx context.Context,
	wctx sdk.Context,
	keeper appkeeper.Keeper,
	srv apptypes.MsgServer,
	appAddr, gatewayAddr string,
) {
	// Prepare the delegation message
	delegateMsg := &apptypes.MsgDelegateToGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Delegate the application to the gateway
	_, err := srv.DelegateToGateway(ctx, delegateMsg)
	require.NoError(t, err)
}

func undelegateAppFromGateway(
	t *testing.T,
	ctx context.Context,
	wctx sdk.Context,
	keeper appkeeper.Keeper,
	srv apptypes.MsgServer,
	appAddr, gatewayAddr string,
) {
	// Prepare the undelegation message
	undelegateMsg := &apptypes.MsgUndelegateFromGateway{
		AppAddress:     appAddr,
		GatewayAddress: gatewayAddr,
	}

	// Undelegate the application from the gateway
	_, err := srv.UndelegateFromGateway(ctx, undelegateMsg)
	require.NoError(t, err)
}
