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

	// duplicates pkg/client/delegation/client.go's del for testing purposes
	delegationEventQuery = "tm.event='Tx' AND message.action='pocket.application.EventDelegateeChange'"
)

func TestDelegationClient(t *testing.T) {
	var (
		expectedAddress         = sample.AccAddress()
		expectedDelegationEvent = apptypes.EventDelegateeChange{
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
	delegationClient, err := delegation.NewDelegationClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)
	require.NotNil(t, delegationClient)

	tests := []struct {
		name string
		fn   func() client.DelegateeChange
	}{
		{
			name: "LastNEvent successfully returns latest delegatee change",
			fn: func() client.DelegateeChange {
				lastDelegateeChange := delegationClient.LastNEvents(ctx, 1)[0]
				return lastDelegateeChange
			},
		},
		{
			name: "EventsSequence successfully returns latest delegatee change",
			fn: func() client.DelegateeChange {
				delegateeChangeObs := delegationClient.EventsSequence(ctx)
				require.NotNil(t, delegateeChangeObs)

				// Ensure that the observable is replayable via Last.
				lastDelegateeChange := delegateeChangeObs.Last(ctx, 1)[0]
				require.Equal(t, expectedAddress, lastDelegateeChange.AppAddress())

				return lastDelegateeChange
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualDelegateeChangeCh := make(chan client.DelegateeChange, 10)

			// Run test functions asynchronously because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() client.DelegateeChange) {
				actualDelegateeChangeCh <- fn()
				close(actualDelegateeChangeCh)
			}(tt.fn)

			select {
			case actualDelegateeChange := <-actualDelegateeChangeCh:
				require.Equal(t, expectedAddress, actualDelegateeChange.AppAddress())
			case <-time.After(testTimeoutDuration):
				t.Fatal("timed out waiting for block event")
			}
		})
	}

	delegationClient.Close()
}
