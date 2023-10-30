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

// NewLocalnetClient returns a new events query client which is configured to
// connect to the localnet sequencer.
func NewLocalnetClient(t *testing.T, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return eventsquery.NewEventsQueryClient(testclient.CometLocalWebsocketURL, opts...)
}

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
