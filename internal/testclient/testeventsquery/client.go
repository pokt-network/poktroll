package testeventsquery

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/internal/mocks/mockclient"
	"github.com/pokt-network/poktroll/internal/testclient"
	"github.com/pokt-network/poktroll/pkg/client"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// NewLocalnetClient returns a new events query client which is configured to
// connect to the localnet sequencer.
func NewLocalnetClient(t *testing.T, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return eventsquery.NewEventsQueryClient(testclient.CometLocalWebsocketURL, opts...)
}

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
		EventsBytes(gomock.AssignableToTypeOf(ctx), gomock.AssignableToTypeOf("")).
		DoAndReturn(
			func(ctx context.Context, query string) (client.EventsBytesObservable, error) {
				require.Equal(t, expectedQuery, query)

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
