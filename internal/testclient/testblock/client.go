package testblock

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"pocket/internal/testclient"
	"pocket/internal/testclient/testquery"
	"pocket/pkg/client"
	"pocket/pkg/client/block"
)

func NewLocalnetClient(ctx context.Context, t *testing.T) client.BlockClient {
	t.Helper()

	queryClient := testquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	bClient, err := block.NewBlockClient(ctx, deps, testclient.CometWebsocketURL)
	require.NoError(t, err)
	return bClient
}
