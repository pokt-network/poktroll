package delegation_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

const (
	testTimeoutDuration = 100 * time.Millisecond

	// duplicates pkg/client/delegation/client.go's delegationEventQuery for testing purposes.
	delegationEventQuery = "message.action='pocket.application.EventRedelegation'"
)

func TestDelegationClient(t *testing.T) {
	var (
		expectedAddress         = sample.AccAddress()
		expectedDelegationEvent = apptypes.EventRedelegation{
			AppAddress: expectedAddress,
		}
		ctx = context.Background()
	)

	expectedEventBz, err := json.Marshal(expectedDelegationEvent)
	require.NoError(t, err)

	eventsQueryClient := testeventsquery.NewAnyTimesEventsBytesEventsQueryClient(
		ctx, t,
		delegationEventQuery,
		expectedEventBz,
	)

	deps := depinject.Supply(eventsQueryClient)

	// Set up delegation client.
	// NB: the URL passed to `NewDelegationClient` is irrelevant here because `eventsQueryClient` is a mock.
	delegationClient, err := delegation.NewDelegationClient(ctx, deps, testclient.CometLocalWebsocketURL)
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
				require.Equal(t, expectedAddress, lastRedelegation.AppAddress())

				return lastRedelegation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualRedelegationCh := make(chan client.Redelegation, 10)

			// Run test functions asynchronously because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() client.Redelegation) {
				actualRedelegationCh <- fn()
				close(actualRedelegationCh)
			}(tt.fn)

			select {
			case actualRedelegation := <-actualRedelegationCh:
				require.Equal(t, expectedAddress, actualRedelegation.AppAddress())
			case <-time.After(testTimeoutDuration):
				t.Fatal("timed out waiting for redelegation event")
			}
		})
	}

	delegationClient.Close()
}
