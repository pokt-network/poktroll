package testquery

import (
	"testing"

	"pocket/internal/testclient"
	"pocket/pkg/client"
	"pocket/pkg/client/query"
)

func NewLocalnetClient(t *testing.T) client.QueryClient {
	t.Helper()

	return query.NewQueryClient(testclient.CometWebsocketURL)
}
