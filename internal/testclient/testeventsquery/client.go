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

// NewLocalnetClient creates and returns a new events query client that connects to the
// localnet sequencer. It leverages the NewEventsQueryClient from the eventsquery package.
//
// Parameters:
// t: Testing object which is passed for marking helper.
// opts: Any additional options to configure the events query client.
//
// Returns: An instance of EventsQueryClient which connects to the localnet sequencer.
func NewLocalnetClient(t *testing.T, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return eventsquery.NewEventsQueryClient(testclient.CometLocalWebsocketURL, opts...)
}

// NewOneTimeEventsQuery creates a mock of the EventsQueryClient which expects a single call to the
// EventsBytes method. It returns a mock client that emits event bytes to the provided publish channel.
//
// Parameters:
// ctx: Context object to define the context for the operation.
// t: Testing object.
// query: The query string for which event bytes are fetched.
// publishCh: Channel to which the event bytes are published.
//
// Returns: A mock instance of EventsQueryClient which behaves as described.
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

// NewOneTimeTxEventsQueryClient initializes a new MockEventsQueryClient that is set up to
// query for transaction events for a specific message sender. This is useful for tests where
// you want to listen for a one-time event related to a specific sender.
//
// Parameters:
// - ctx: The context to pass to the client.
// - t: The testing.T instance for assertions.
// - key: The keyring record from which the signing address is derived.
// - publishCh: A channel where the events are published.
//
// Returns:
// - A new instance of mockclient.MockEventsQueryClient set up for the specific query.
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
