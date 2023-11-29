package testdelegation

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

// NewLocalnetClient creates and returns a new DelegationClient that's configured for
// use with the localnet sequencer.
func NewLocalnetClient(ctx context.Context, t *testing.T) client.DelegationClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	dClient, err := delegation.NewDelegationClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)

	return dClient
}

// NewAnyTimesDelegateeChangesSequence creates a new mock DelegationClient.
// This mock DelegationClient will expect any number of calls to DelegateeChangesSequence,
// and when that call is made, it returns the given EventsObservable[DelegateeChange].
func NewAnyTimesDelegateeChangesSequence(
	t *testing.T,
	delegateeChangeObs observable.Observable[client.DelegateeChange],
) *mockclient.MockDelegationClient {
	t.Helper()

	// Create a mock for the delegation client which expects the
	// LastNDelegateeChanges method to be called any number of times.
	delegationClientMock := NewAnyTimeLastNDelegateeChangesClient(t, "")

	// Set up the mock expectation for the DelegateeChangesSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// delegation events sent on the given delegateeChangeObs.
	delegationClientMock.EXPECT().
		DelegateeChangesSequence(
			gomock.AssignableToTypeOf(context.Background()),
		).
		Return(delegateeChangeObs).
		AnyTimes()

	return delegationClientMock
}

// NewOneTimeDelegateeChangesSequenceDelegationClient creates a new mock
// DelegationClient. This mock DelegationClient will expect a call to
// DelegateeChangesSequence, and when that call is made, it returns a new
// DelegateeChangeReplayObservable that publishes DelegateeChange events sent on
// the given delegateeChangesPublishCh.
// delegateeChangesPublishCh is the channel the caller can use to publish
// DelegateeChange events to the observable.
func NewOneTimeDelegateeChangesSequenceDelegationClient(
	t *testing.T,
	delegateeChangesPublishCh chan client.DelegateeChange,
) *mockclient.MockDelegationClient {
	t.Helper()

	// Create a mock for the delegation client which expects the
	// LastNDelegateeChanges method to be called any number of times.
	delegationClientMock := NewAnyTimeLastNDelegateeChangesClient(t, "")

	// Set up the mock expectation for the DelegateeChangesSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// delegation changes sent on the given delegateeChangesPublishCh.
	delegationClientMock.EXPECT().DelegateeChangesSequence(
		gomock.AssignableToTypeOf(context.Background()),
	).DoAndReturn(func(ctx context.Context) client.DelegateeChangeReplayObservable {
		// Create a new replay observable with a replay buffer size of 1.
		// DelegateeChange events are published to this observable via the
		// provided delegateeChangesPublishCh.
		withPublisherOpt := channel.WithPublisher(delegateeChangesPublishCh)
		obs, _ := channel.NewReplayObservable[client.DelegateeChange](
			ctx, 1, withPublisherOpt,
		)
		return obs
	})

	return delegationClientMock
}

// NewAnyTimeLastNDelegateeChangesClient creates a mock DelegationClient that
// expects calls to the LastNDelegateeChanges method any number of times. When
// the LastNDelegateeChanges method is called, it returns a mock DelegateeChange
// with the provided appAddress.
func NewAnyTimeLastNDelegateeChangesClient(
	t *testing.T,
	appAddress string,
) *mockclient.MockDelegationClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock delegateeChange that returns the provided appAddress
	delegateeChange := NewAnyTimesDelegateeChange(t, appAddress)
	// Create a mock delegation client that expects calls to
	// LastNDelegateeChanges method and returns the mock delegateeChange.
	delegationClientMock := mockclient.NewMockDelegationClient(ctrl)
	delegationClientMock.EXPECT().
		LastNDelegateeChanges(gomock.Any(), gomock.Any()).
		Return([]client.DelegateeChange{delegateeChange}).AnyTimes()

	return delegationClientMock
}

// NewAnyTimesDelegateeChange creates a mock DelegateeChange that expects calls
// to the AppAddress method any number of times. When the method is called, it
// returns the provided app address.
func NewAnyTimesDelegateeChange(
	t *testing.T,
	appAddress string,
) *mockclient.MockDelegateeChange {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock delegateeChange that returns the provided address AnyTimes.
	delegateeChange := mockclient.NewMockDelegateeChange(ctrl)
	delegateeChange.EXPECT().AppAddress().Return(appAddress).AnyTimes()

	return delegateeChange
}
