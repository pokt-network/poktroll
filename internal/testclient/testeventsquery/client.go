// Package testeventsquery provides helper methods and mocks for testing events query functionality.
package testeventsquery

import (
	"context"
	"fmt"
	"testing"

	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/internal/testclient"
	"pocket/pkg/client"
	eventsquery "pocket/pkg/client/events_query"
	"pocket/pkg/either"
	"pocket/pkg/observable/channel"
)

// NewLocalnetClient creates and returns a new events query client that configured
// for use with the localnet sequencer. Any options provided are applied to the client.
func NewLocalnetClient(t *testing.T, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return eventsquery.NewEventsQueryClient(testclient.CometLocalWebsocketURL, opts...)
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
	publishCh *chan<- either.Bytes,
) *mockclient.MockEventsQueryClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	eventsQueryClient := mockclient.NewMockEventsQueryClient(ctrl)
	eventsQueryClient.EXPECT().EventsBytes(gomock.Eq(ctx), gomock.Eq(query)).
		DoAndReturn(func(
			ctx context.Context,
			query string,
		) (eventsBzObservable client.EventsBytesObservable, err error) {
			eventsBzObservable, *publishCh = channel.NewObservable[either.Bytes]()
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
		publishCh,
	)
}
