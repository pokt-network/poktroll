package testeventsquery

import (
	"testing"

	"pocket/internal/testclient"
	"pocket/pkg/client"
	eventsquery "pocket/pkg/client/events_query"
)

func NewLocalnetClient(t *testing.T, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return eventsquery.NewEventsQueryClient(testclient.CometWebsocketURL, opts...)
}
