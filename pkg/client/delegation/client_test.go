package delegation_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

const (
	testTimeoutDuration = 100 * time.Millisecond

	// duplicates pkg/client/delegation/client.go's delegationEventQuery for testing purposes.
	delegationEventQuery = "tm.event='Tx' AND message.module='application'"
)

func TestDelegationClient(t *testing.T) {
	var (
		expectedAppAddress     = sample.AccAddress()
		expectedGatewayAddress = sample.AccAddress()
		ctx                    = context.Background()
	)

	expectedEventBz := testdelegation.NewRedelegationEventBytes(t, expectedAppAddress, expectedGatewayAddress)

	eventsQueryClient := testeventsquery.NewAnyTimesEventsBytesEventsQueryClient(
		ctx, t,
		delegationEventQuery,
		expectedEventBz,
	)

	deps := depinject.Supply(eventsQueryClient)

	// Set up delegation client.
	// NB: the URL passed to `NewDelegationClient` is irrelevant here because `eventsQueryClient` is a mock.
	delegationClient, err := delegation.NewDelegationClient(ctx, deps)
	require.NoError(t, err)
	require.NotNil(t, delegationClient)

	tests := []struct {
		name string
		fn   func() *apptypes.EventRedelegation
	}{
		{
			name: "LastNRedelegations successfully returns latest redelegation",
			fn: func() *apptypes.EventRedelegation {
				lastRedelegation := delegationClient.LastNRedelegations(ctx, 1)[0]
				require.Equal(t, expectedAppAddress, lastRedelegation.GetApplication().GetAddress())
				require.Contains(t, lastRedelegation.GetApplication().GetDelegateeGatewayAddresses(), expectedGatewayAddress)
				return lastRedelegation
			},
		},
		{
			name: "RedelegationsSequence successfully returns latest redelegation",
			fn: func() *apptypes.EventRedelegation {
				redelegationObs := delegationClient.RedelegationsSequence(ctx)
				require.NotNil(t, redelegationObs)

				// Ensure that the observable is replayable via Last.
				lastRedelegation := redelegationObs.Last(ctx, 1)[0]
				require.Equal(t, expectedAppAddress, lastRedelegation.GetApplication().GetAddress())
				require.Contains(t, lastRedelegation.GetApplication().GetDelegateeGatewayAddresses(), expectedGatewayAddress)

				return lastRedelegation
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualRedelegationCh := make(chan *apptypes.EventRedelegation, 10)

			// Run test functions asynchronously because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() *apptypes.EventRedelegation) {
				actualRedelegationCh <- fn()
				close(actualRedelegationCh)
			}(test.fn)

			select {
			case actualRedelegation := <-actualRedelegationCh:
				require.Equal(t, expectedAppAddress, actualRedelegation.GetApplication().GetAddress())
				require.Contains(t, actualRedelegation.GetApplication().GetDelegateeGatewayAddresses(), expectedGatewayAddress)
			case <-time.After(testTimeoutDuration):
				t.Fatal("timed out waiting for redelegation event")
			}
		})
	}

	delegationClient.Close()
}
