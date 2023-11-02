package testeventsquery

import (
	"testing"

	"github.com/pokt-network/poktroll/internal/testclient"
	"github.com/pokt-network/poktroll/pkg/client"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
)

// NewLocalnetClient returns a new events query client which is configured to
// connect to the localnet sequencer.
func NewLocalnetClient(t *testing.T, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	t.Helper()

	return eventsquery.NewEventsQueryClient(testclient.CometLocalWebsocketURL, opts...)
}
