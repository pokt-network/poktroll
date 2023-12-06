//go:build integration

package delegation_test

// TODO(@h5law): Figure out how to use real components of the localnet
//  - Create app and gateway actors
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

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/testutil/network"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

const (
	delegationIntegrationSubTimeout = 180 * time.Second
)

// TODO_UPNEXT(@h5law): Figure out the correct way to subscribe to events on the
// simulated localnet. Currently this test doesn't work. Although the block client
// subscribes it doesn't receive any events.
func TestDelegationClient_RedelegationsObservables(t *testing.T) {
	t.SkipNow()
	// Create the network with 2 applications and 1 gateway
	net, appAddresses, gatewayAddr := createNetworkWithApplicationsAndGateways(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the delegation client
	evtQueryClient := events.NewEventsQueryClient("ws://localhost:26657/websocket")
	deps := depinject.Supply(evtQueryClient)
	delegationClient, err := delegation.NewDelegationClient(ctx, deps, "ws://localhost:26657/websocket")
	require.NoError(t, err)
	require.NotNil(t, delegationClient)
	t.Cleanup(func() {
		delegationClient.Close()
	})

	// Subscribe to the delegation events
	delegationSub := delegationClient.RedelegationsSequence(ctx).Subscribe(ctx)

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
		// Redelegation event alternates between app1 and app2
		var previousRedelegation client.Redelegation
		for change := range delegationSub.Ch() {
			t.Logf("received delegation change: %+v", change)
			// Verify that the Redelegation event is valid and that the address
			// of the Redelegation event alternates between app1 and app2
			if previousRedelegation != nil {
				require.NotEqual(t, previousRedelegation.AppAddress(), change.AppAddress())
				if previousRedelegation.AppAddress() == appAddresses[0] {
					require.Equal(t, appAddresses[1], change.AppAddress())
				} else {
					require.Equal(t, appAddresses[0], change.AppAddress())
				}
			}
			previousRedelegation = change

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

	// Delegate from app1 to gateway
	t.Log(time.Now().String())
	t.Logf("delegating from app %s to gateway %s", appAddresses[0], gatewayAddr)
	network.DelegateAppToGateway(t, net, appAddresses[0], gatewayAddr)
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())
	// Delegate from app2 to gateway
	t.Logf("delegating from app %s to gateway %s", appAddresses[1], gatewayAddr)
	network.DelegateAppToGateway(t, net, appAddresses[1], gatewayAddr)
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())
	// Undelegate from app1 to gateway
	t.Logf("undelegating from app %s to gateway %s", appAddresses[0], gatewayAddr)
	network.UndelegateAppFromGateway(t, net, appAddresses[0], gatewayAddr)
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())
	// Undelegate from app2 to gateway
	t.Logf("undelegating from app %s to gateway %s", appAddresses[1], gatewayAddr)
	network.UndelegateAppFromGateway(t, net, appAddresses[1], gatewayAddr)
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equal(t, expectedChanges, delegationChangeCounter)
	case <-time.After(delegationIntegrationSubTimeout):
		t.Log(time.Now().String())
		t.Fatalf(
			"timed out waiting for delegation subscription; expected %d delegation events, got %d",
			expectedChanges, delegationChangeCounter,
		)
	}
}

// createNetworkWithApplicationsAndGateways creates a network with 2 applications
// and 1 gateway. It returns the network with all accoutns initialised via a
// transaction from the first validator.
func createNetworkWithApplicationsAndGateways(
	t *testing.T,
) (net *network.Network, appAddresses []string, gatewayAddress string) {
	// Prepare the network
	cfg := network.DefaultConfig()
	net = network.New(t, cfg)
	ctx := net.Validators[0].ClientCtx

	// Prepare the keyring for the 2 applications and 1 gateway account
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 3)
	ctx = ctx.WithKeyring(kr)

	// Initialise all the accounts
	for i, account := range accounts {
		signatureSequenceNumber := i + 1
		network.InitAccountWithSequence(t, net, account.Address, signatureSequenceNumber)
	}
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())

	addresses := make([]string, len(accounts))
	for i, account := range accounts {
		addresses[i] = account.Address.String()
	}

	// Create two applications
	appGenesisState := network.ApplicationModuleGenesisStateWithAccounts(t, addresses[0:2])
	buf, err := cfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[apptypes.ModuleName] = buf

	// Create a single gateway
	gatewayGenesisState := network.GatewayModuleGenesisStateWithAccounts(t, addresses[2:3])
	buf, err = cfg.Codec.MarshalJSON(gatewayGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[gatewaytypes.ModuleName] = buf

	return net, addresses[0:2], addresses[2]
}
