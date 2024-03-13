package delegation_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

// TODO_IN_THIS_PR: Keep looking at the tests in this file.

const (
	testTimeoutDuration = 1 * time.Second

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
		fn   func() client.Redelegation
	}{
		{
			name: "LastNRedelegations successfully returns latest redelegation",
			fn: func() client.Redelegation {
				lastRedelegation := delegationClient.LastNRedelegations(ctx, 1)[0]
				require.Equal(t, expectedAppAddress, lastRedelegation.GetAppAddress())
				require.Equal(t, expectedGatewayAddress, lastRedelegation.GetGatewayAddress())
				return lastRedelegation
			},
		},
		{
			name: "RedelegationsSequence successfully returns latest redelegation",
			fn: func() client.Redelegation {
				redelegationObs := delegationClient.RedelegationsSequence(ctx)
				require.NotNil(t, redelegationObs)

				// Ensure that the observable is replayable via Last.
				lastRedelegation := redelegationObs.Last(ctx, 1)[0]
				require.Equal(t, expectedAppAddress, lastRedelegation.GetAppAddress())
				require.Equal(t, expectedGatewayAddress, lastRedelegation.GetGatewayAddress())

				return lastRedelegation
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualRedelegationCh := make(chan client.Redelegation, 10)
			// Run test functions asynchronously because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() client.Redelegation) {
				actualRedelegationCh <- fn()
				close(actualRedelegationCh)
			}(test.fn)

			select {
			case actualRedelegation := <-actualRedelegationCh:
				require.Equal(t, expectedAppAddress, actualRedelegation.GetAppAddress())
				require.Equal(t, expectedGatewayAddress, actualRedelegation.GetGatewayAddress())
			case <-time.After(testTimeoutDuration):
				t.Fatal("timed out waiting for redelegation event")
			}
		})
	}

	delegationClient.Close()
}
