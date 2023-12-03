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
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

// NewLocalnetClient creates and returns a new DelegationClient that's configured for
// use with the localnet sequencer.
func NewLocalnetClient(ctx context.Context, t *testing.T) client.DelegationClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	dClient, err := delegation.NewDelegationClient(ctx, deps)
	require.NoError(t, err)

	return dClient
}

// NewAnyTimesRedelegationsSequence creates a new mock DelegationClient.
// This mock DelegationClient will expect any number of calls to RedelegationsSequence,
// and when that call is made, it returns the given EventsObservable[Redelegation].
func NewAnyTimesRedelegationsSequence(
	t *testing.T,
	redelegationObs observable.Observable[client.Redelegation],
) *mockclient.MockDelegationClient {
	t.Helper()

	// Create a mock for the delegation client which expects the
	// LastNRedelegations method to be called any number of times.
	delegationClientMock := NewAnyTimeLastNRedelegationsClient(t, "")

	// Set up the mock expectation for the RedelegationsSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// redelegation events sent on the given redelegationObs.
	delegationClientMock.EXPECT().
		RedelegationsSequence(
			gomock.AssignableToTypeOf(context.Background()),
		).
		Return(redelegationObs).
		AnyTimes()

	return delegationClientMock
}

// NewOneTimeRedelegationsSequenceDelegationClient creates a new mock
// DelegationClient. This mock DelegationClient will expect a call to
// RedelegationsSequence, and when that call is made, it returns a new
// RedelegationReplayObservable that publishes Redelegation events sent on
// the given redelegationPublishCh.
// redelegationPublishCh is the channel the caller can use to publish
// Redelegation events to the observable.
func NewOneTimeRedelegationsSequenceDelegationClient(
	t *testing.T,
	redelegationPublishCh chan client.Redelegation,
) *mockclient.MockDelegationClient {
	t.Helper()

	// Create a mock for the delegation client which expects the
	// LastNRedelegations method to be called any number of times.
	delegationClientMock := NewAnyTimeLastNRedelegationsClient(t, "")

	// Set up the mock expectation for the RedelegationsSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// delegation changes sent on the given redelegationPublishCh.
	delegationClientMock.EXPECT().RedelegationsSequence(
		gomock.AssignableToTypeOf(context.Background()),
	).DoAndReturn(func(ctx context.Context) client.RedelegationReplayObservable {
		// Create a new replay observable with a replay buffer size of 1.
		// Redelegation events are published to this observable via the
		// provided redelegationPublishCh.
		withPublisherOpt := channel.WithPublisher(redelegationPublishCh)
		obs, _ := channel.NewReplayObservable[client.Redelegation](
			ctx, 1, withPublisherOpt,
		)
		return obs
	})

	delegationClientMock.EXPECT().Close().AnyTimes()

	return delegationClientMock
}

// NewAnyTimeLastNRedelegationsClient creates a mock DelegationClient that
// expects calls to the LastNRedelegations method any number of times. When
// the LastNRedelegations method is called, it returns a mock Redelegation
// with the provided appAddress.
func NewAnyTimeLastNRedelegationsClient(
	t *testing.T,
	appAddress string,
) *mockclient.MockDelegationClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock redelegation that returns the provided appAddress
	redelegation := NewAnyTimesRedelegation(t, appAddress)
	// Create a mock delegation client that expects calls to
	// LastNRedelegations method and returns the mock redelegation.
	delegationClientMock := mockclient.NewMockDelegationClient(ctrl)
	delegationClientMock.EXPECT().
		LastNRedelegations(gomock.Any(), gomock.Any()).
		Return([]client.Redelegation{redelegation}).AnyTimes()

	return delegationClientMock
}

// NewAnyTimesRedelegation creates a mock Redelegation that expects calls
// to the AppAddress method any number of times. When the method is called, it
// returns the provided app address.
func NewAnyTimesRedelegation(
	t *testing.T,
	appAddress string,
) *mockclient.MockRedelegation {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock redelegation that returns the provided address AnyTimes.
	redelegation := mockclient.NewMockRedelegation(ctrl)
	redelegation.EXPECT().GetAppAddress().Return(appAddress).AnyTimes()

	return redelegation
}
