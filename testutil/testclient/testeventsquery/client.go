package testeventsquery

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient"
)

// NewLocalnetClient creates and returns a new events query client that's configured
// for use with the LocalNet validator. Any options provided are applied to the client.
func NewLocalnetClient(t gocuke.TestingT, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return events.NewEventsQueryClient(testclient.CometLocalWebsocketURL, opts...)
}

// NewOneTimeEventsQuery creates a mock of the EventsQueryClient which expects
// a single call to the EventsBytes method. It returns a mock client whose event
// bytes method always  constructs a new observable. query is the query string
// for which event bytes subscription is expected to be for.
// The caller can simulate blockchain events by sending on publishCh, the value
// of which is set to the publish channel of the events bytes observable publish
// channel.
func NewOneTimeEventsQuery(
	ctx context.Context,
	t *testing.T,
	query string,
	publishChMu *sync.Mutex,
	publishCh *chan<- either.Bytes,
) *mockclient.MockEventsQueryClient {
	t.Helper()
	ctrl := gomock.NewController(t)
	eventsQueryClient := mockclient.NewMockEventsQueryClient(ctrl)
	eventsQueryClient.EXPECT().EventsBytes(gomock.Any(), gomock.Eq(query)).
		DoAndReturn(func(
			ctx context.Context,
			query string,
		) (eventsBzObservable client.EventsBytesObservable, err error) {
			publishChMu.Lock()
			eventsBzObservable, *publishCh = channel.NewObservable[either.Bytes]()
			publishChMu.Unlock()
			return eventsBzObservable, nil
		}).Times(1)
	return eventsQueryClient
}

// NewOneTimeTxEventsQueryClient creates a mock of the Events that expects to to
// a single call to the EventsBytes method where the query is for transaction
// events for sender address matching that of the given key.
// The caller can simulate blockchain events by sending on publishCh, the value
// of which is set to the publish channel of the events bytes observable publish
// channel.
func NewOneTimeTxEventsQueryClient(
	ctx context.Context,
	t *testing.T,
	key *cosmoskeyring.Record,
	publishChMu *sync.Mutex,
	publishCh *chan<- either.Bytes,
) *mockclient.MockEventsQueryClient {
	t.Helper()

	signingAddr, err := key.GetAddress()
	require.NoError(t, err)

	expectedEventsQuery := fmt.Sprintf(
		"tm.event='Tx' AND message.sender='%s'",
		signingAddr,
	)
	return NewOneTimeEventsQuery(
		ctx, t,
		expectedEventsQuery,
		publishChMu,
		publishCh,
	)
}

// NewAnyTimesEventsBytesEventsQueryClient returns a new events query client which
// is configured to return the expected event bytes when queried with the expected
// query, any number of times. The returned client also expects to be closed once.
func NewAnyTimesEventsBytesEventsQueryClient(
	ctx context.Context,
	t *testing.T,
	expectedQuery string,
	expectedEventBytes []byte,
) client.EventsQueryClient {
	t.Helper()

	ctrl := gomock.NewController(t)
	eventsQueryClient := mockclient.NewMockEventsQueryClient(ctrl)
	eventsQueryClient.EXPECT().Close().Times(1)
	eventsQueryClient.EXPECT().
		EventsBytes(gomock.Any(), gomock.Eq(expectedQuery)).
		DoAndReturn(
			func(ctx context.Context, query string) (client.EventsBytesObservable, error) {
				bytesObsvbl, bytesPublishCh := channel.NewReplayObservable[either.Bytes](ctx, 1)

				// Now that the observable is set up, publish the expected event bytes.
				// Only need to send once because it's a ReplayObservable.
				bytesPublishCh <- either.Success(expectedEventBytes)

				// Wait a tick for the observables to be set up. This isn't strictly
				// necessary but is done to mitigate test flakiness.
				time.Sleep(10 * time.Millisecond)

				return bytesObsvbl, nil
			},
		).
		AnyTimes()

	return eventsQueryClient
}
